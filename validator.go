package aero

type Validator interface {
	Validate(i any) error
}
