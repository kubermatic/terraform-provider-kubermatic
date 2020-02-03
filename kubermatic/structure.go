package kubermatic

func int32ToPtr(i int32) *int32 {
	return &i
}

func int64ToPtr(i int) *int64 {
	v := int64(i)
	return &v
}

func strToPtr(s string) *string {
	return &s
}
