package parts

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

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
	"RTreadLn": {FunLiteral, NativeMethod{
		Args: []string{},
		Body: func(vm *VM, args []*Literal) error {
			reader := bufio.NewReader(os.Stdin)
			text, err := reader.ReadString('\n')

			if err != nil {
				println(err.Error())
			}

			vm.ReturnValue = &Literal{StringLiteral, text}
			return nil
		},
	}},
	"RTArray": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTHas": {FunLiteral, NativeMethod{
				Args: []string{"arr", "elt"},
				Body: func(vm *VM, args []*Literal) error {
					if len(args) < 2 {
						return nil
					}

					acc := args[0]

					if acc.LiteralType != ParsedListLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					for _, val := range acc.Value.(PartsObject).Entries {
						if val, err := args[1].opEq(val); err != nil && val.Value.(bool) {
							vm.ReturnValue = &Literal{BoolLiteral, true}
							return nil
						}
					}

					vm.ReturnValue = &Literal{BoolLiteral, false}
					return nil
				},
			}},
			"RTAppendAll": {FunLiteral, NativeMethod{
				Args: []string{"arr", "other"},
				Body: func(vm *VM, args []*Literal) error {
					if len(args) < 2 {
						return nil
					}

					arr := args[0]

					if arr.LiteralType != ParsedListLiteral {
						panic("expected array type (base array)")
					}

					other := args[1]

					if other.LiteralType != ParsedListLiteral {
						panic("expected array type (2nd arg)")
					}

					entries := other.Value.(PartsIndexable).GetAll()

					keys := make([]string, 0, len(entries))

					for k := range entries {
						keys = append(keys, k)
					}

					sort.Slice(keys, func(i, j int) bool {

						numStrI := strings.TrimPrefix(keys[i], "IT")
						numI, errI := strconv.Atoi(numStrI)

						if errI != nil {
							panic(fmt.Sprintf("Warning: Could not parse number from key %s: %v\n", keys[i], errI))
						}

						// Extract the numerical part from key j
						numStrJ := strings.TrimPrefix(keys[j], "IT")
						numJ, errJ := strconv.Atoi(numStrJ)
						if errJ != nil {
							panic(fmt.Sprintf("Warning: Could not parse number from key %s: %v\n", keys[j], errJ))
						}

						return numI < numJ
					})

					for _, key := range keys {
						arr.Value.(PartsIndexable).SetByKey(fmt.Sprintf("IT%d", arr.Value.(PartsIndexable).Length()), entries[key])
					}

					vm.ReturnValue = arr
					return nil
				},
			}},
		},
	}},
	"RTObject": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTHas": {FunLiteral, NativeMethod{
				Args: []string{"obj", "elt"},
				Body: func(vm *VM, args []*Literal) error {
					if len(args) < 2 {
						return nil
					}

					acc := args[0]

					if acc.LiteralType != ParsedObjLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					if args[1].LiteralType != StringLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					casted, ok := acc.Value.(PartsIndexable)

					if !ok {
						panic(reflect.ValueOf(acc.Value).Type().String())
					}

					vm.ReturnValue = &Literal{BoolLiteral, casted.HasByKey(fmt.Sprintf("RT%s", args[1].Value.(string)))}
					return nil
				},
			}},
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
