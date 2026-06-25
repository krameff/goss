package resource

import "reflect"

// YAMLKey returns the gossfile map key for a resource.
func YAMLKey(res Resource) string {
	value := reflect.ValueOf(res)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	field := value.FieldByName("id")
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	if rr, ok := res.(ResourceRead); ok {
		return rr.ID()
	}
	return ""
}

// Ref returns the canonical dependency reference for a resource.
func Ref(res Resource) string {
	return res.TypeKey() + ":" + YAMLKey(res)
}
