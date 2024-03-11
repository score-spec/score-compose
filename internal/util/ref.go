package util

func Ref[k any](input k) *k {
	return &input
}

func DerefOr[k any](input *k, def k) k {
	if input == nil {
		return def
	}
	return *input
}
