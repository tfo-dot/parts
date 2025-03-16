package parts

import "fmt"

var StandardLibrary = map[string]*Literal{
	"RTprint": {FunLiteral, NativeMethod{
		Args: []string{"arg"},
		Body: func(vm *VM, args []*Literal) {
			if len(args) == 0 {
				return
			}

			fmt.Print(args[0].pretify())
		},
	}},
	"RTprintLn": {FunLiteral, NativeMethod{
		Args: []string{"arg"},
		Body: func(vm *VM, args []*Literal) {
			if len(args) == 0 {
				return
			}

			fmt.Println(args[0].pretify())
		},
	}},
}

type NativeMethod struct {
	Args []string
	Body      func(vm *VM, args []*Literal)
}

func (m NativeMethod) Call(vm *VM) {
	arguments := make([]*Literal, len(m.Args))

	for idx, key := range m.Args {
		val, err :=  vm.Enviroment.resolve(fmt.Sprintf("RT%s", key))

		if err != nil {
			panic(err)
		}

		arguments[idx] = val 
	}

	m.Body(vm, arguments)
}

func (m NativeMethod) GetArguments() []string {
	return m.Args
}