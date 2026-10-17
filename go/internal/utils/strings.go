package utils

func ToStrings[T ~string](values []T) []string {
	converted := make([]string, len(values))

	for i, value := range values {
		converted[i] = string(value)
	}

	return converted
}
