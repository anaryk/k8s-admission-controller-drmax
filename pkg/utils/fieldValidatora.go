package utils

import "strconv"

func ValidateIntFiled(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

func ValidateStringField(value string) bool {
	return value != ""
}
