package parts

import "fmt"

var StandardLibrary = map[string]*Literal{
	"RTprint": {FunLiteral, NativeMethod{
		Args: []string{"arg"},
		Body: func(vm *VM, args []*Literal) error {
			if len(args) == 0 {
				return nil
			}

			fmt.Print(args[0].pretify())
			return nil
		},
	}},
	"RTprintLn": {FunLiteral, NativeMethod{
		Args: []string{"arg"},
		Body: func(vm *VM, args []*Literal) error {
			if len(args) == 0 {
				return nil
			}

			fmt.Println(args[0].pretify())
			return nil
		},
	}},
}

type NativeMethod struct {
	Args []string
	Body func(vm *VM, args []*Literal) error
}

func (m NativeMethod) Call(vm *VM) error {
	arguments := make([]*Literal, len(m.Args))

	for idx, key := range m.Args {
		val, err := vm.Enviroment.resolve(fmt.Sprintf("RT%s", key))

		if err != nil {
			return err
		}

		arguments[idx] = val
	}

	return m.Body(vm, arguments)
}

func (m NativeMethod) GetArguments() []string {
	return m.Args
}
