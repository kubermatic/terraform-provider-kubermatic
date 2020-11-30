package metakube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/go-openapi/runtime"
	oclient "github.com/go-openapi/runtime/client"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/mitchellh/go-homedir"
	k8client "github.com/syseleven/terraform-provider-metakube/go-metakube/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// wait this time before starting resource checks
	requestDelay = time.Second
	// smallest time to wait before refreshes
	retryTimeout = time.Second
)

type metakubeProviderMeta struct {
	client *k8client.MetaKube
	auth   runtime.ClientAuthInfoWriter
	log    *zap.SugaredLogger
}

// Provider is a MetaKube Terraform Provider.
func Provider() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_HOST", "https://localhost"),
				Description: "The hostname of MetaKube API (in form of URI)",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_TOKEN", ""),
				Description: "Authentication token",
			},
			"token_path": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{
						"METAKUBE_TOKEN_PATH",
					},
					"~/.metakube/auth"),
				Description: "Path to the MetaKube authentication token, defaults to ~/.metakube/auth",
			},
			"development": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_DEV", false),
				Description: "Run development mode.",
			},
			"debug": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_DEBUG", false),
				Description: "Run debug mode.",
			},
			"log_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("METAKUBE_LOG_PATH", ""),
				Description: "Path to store logs",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"metakube_project":               resourceProject(),
			"metakube_cluster":               resourceCluster(),
			"metakube_node_deployment":       resourceNodeDeployment(),
			"metakube_sshkey":                resourceSSHKey(),
			"metakube_service_account":       resourceServiceAccount(),
			"metakube_service_account_token": resourceServiceAccountToken(),
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
	logDev := d.Get("development").(bool)
	logDebug := d.Get("debug").(bool)
	logPath := d.Get("log_path").(string)
	host := d.Get("host").(string)
	token := d.Get("token").(string)
	tokenPath := d.Get("token_path").(string)
	return newMetaKubeProviderMeta(logDev, logDebug, logPath, host, token, tokenPath, fd)
}

func newMetaKubeProviderMeta(logDev, logDebug bool, logPath, host, token, tokenPath string, fd *os.File) (*metakubeProviderMeta, error) {
	var (
		k   metakubeProviderMeta
		err error
	)

	k.log, err = newLogger(logDev, logDebug, logPath, fd)
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

func newLogger(logDev, logDebug bool, logPath string, fd *os.File) (*zap.SugaredLogger, error) {
	var (
		ec    zapcore.EncoderConfig
		cores []zapcore.Core
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	)

	if logDev || logDebug {
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	if logDev {
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

func newClient(host string) (*k8client.MetaKube, error) {
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

// getErrorResponse converts the client error response to string
func getErrorResponse(err error) string {
	rawData, newErr := json.Marshal(err)
	if newErr != nil {
		return err.Error()
	}
	return string(rawData)
}
