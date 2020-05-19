package kubermatic

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/go-openapi/runtime"
	oclient "github.com/go-openapi/runtime/client"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	k8client "github.com/kubermatic/go-kubermatic/client"
	"github.com/mitchellh/go-homedir"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// wait this time before starting resource checks
	requestDelay = time.Second
	// smallest time to wait before refreshes
	retryTimeout = time.Second
)

type kubermaticProviderMeta struct {
	client *k8client.Kubermatic
	auth   runtime.ClientAuthInfoWriter
	log    *zap.SugaredLogger
}

// Provider is a Kubermatic Terraform Provider.
func Provider() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERMATIC_HOST", "https://localhost"),
				Description: "The Kubermatic hostname",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERMATIC_TOKEN", ""),
				Description: "Authorization token",
			},
			"token_path": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{
						"KUBERMATIC_TOKEN_PATH",
					},
					"~/.kubermatic/auth"),
				Description: "Path to the Kubermatic authorization token, defaults to ~/.kubermatic/auth",
			},
			"development": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERMATIC_DEV", false),
				Description: "Run development mode.",
			},
			"debug": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERMATIC_DEBUG", false),
				Description: "Run debug mode.",
			},
			"log_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBERMATIC_LOG_PATH", ""),
				Description: "Path to store logs",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"kubermatic_project":         resourceProject(),
			"kubermatic_cluster":         resourceCluster(),
			"kubermatic_node_deployment": resourceNodeDeployment(),
		},
	}

	// copying stderr because of https://github.com/hashicorp/go-plugin/issues/93
	// as an example the standard log pkg points to the "old" stderr
	stderr := os.Stderr

	p.ConfigureFunc = func(d *schema.ResourceData) (interface{}, error) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return configure(d, terraformVersion, stderr)
	}

	return p
}

func configure(d *schema.ResourceData, terraformVersion string, fd *os.File) (interface{}, error) {
	var logdev, logdebug bool
	var logpath, host, token, tokenPath string
	if v, ok := d.Get("development").(bool); ok {
		logdev = v
	}
	if v, ok := d.Get("debug").(bool); ok {
		logdebug = v
	}
	if v, ok := d.Get("log_path").(string); ok {
		logpath = v
	}
	if v, ok := d.Get("host").(string); ok {
		host = v
	}
	if v, ok := d.Get("token").(string); ok {
		token = v
	}
	if v, ok := d.Get("token_path").(string); ok {
		tokenPath = v
	}
	return newKubermaticProviderMeta(logdev, logdebug, logpath, host, token, tokenPath, fd)
}

func newKubermaticProviderMeta(logdev, logdebug bool, logpath, host, token, tokenPath string, fd *os.File) (*kubermaticProviderMeta, error) {
	var (
		k   kubermaticProviderMeta
		err error
	)

	k.log, err = newLogger(logdev, logdebug, logpath, fd)
	if err != nil {
		return nil, err
	}

	k.client, err = newClient(host)
	if err != nil {
		return nil, err
	}

	k.auth, err = newAuth(token, tokenPath)
	if err != nil {
		return nil, err
	}

	return &k, nil
}

func newLogger(logdev, logdebug bool, logPath string, fd *os.File) (*zap.SugaredLogger, error) {
	var (
		ec    zapcore.EncoderConfig
		cores []zapcore.Core
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	)

	if logdev || logdebug {
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	if logdev {
		ec = zap.NewDevelopmentEncoderConfig()
		ec.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		ec = zap.NewProductionEncoderConfig()
		ec.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("[" + level.CapitalString() + "]")
		}
	}
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	ec.EncodeDuration = zapcore.StringDurationEncoder

	if logPath != "" {
		jsonEC := ec
		jsonEC.EncodeLevel = zapcore.LowercaseLevelEncoder
		sink, _, err := zap.Open(logPath)
		if err != nil {
			return nil, err
		}
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(jsonEC), sink, level))
	}

	cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(ec), zapcore.AddSync(fd), level))
	core := zapcore.NewTee(cores...)
	return zap.New(core).Sugar(), nil
}

func newClient(host string) (*k8client.Kubermatic, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	return k8client.NewHTTPClientWithConfig(nil, &k8client.TransportConfig{
		Host:     u.Host,
		BasePath: u.Path,
		Schemes:  []string{u.Scheme},
	}), nil
}

func newAuth(token, tokenPath string) (runtime.ClientAuthInfoWriter, error) {
	if token == "" && tokenPath != "" {
		p, err := homedir.Expand(tokenPath)
		if err != nil {
			return nil, err
		}
		rawToken, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}
		token = string(bytes.Trim(rawToken, "\n"))
	} else if token == "" {
		return nil, fmt.Errorf("Missing authorization token")
	}

	return oclient.BearerToken(token), nil
}
