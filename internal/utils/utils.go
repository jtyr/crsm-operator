package utils

import (
	"fmt"
)

func NamespacedName(name, namespace string) string {
	return fmt.Sprintf("%s@%s", name, namespace)
}
