package utils

// ContainString will return true when slice of string contain string
func ContainString(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
