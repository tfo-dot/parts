package parts

import (
	"bufio"
	"errors"
	"fmt"
	"os"
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
						return errors.New("obj is not parts PartsIndexable")
					}

					vm.ReturnValue = &Literal{BoolLiteral, casted.HasByKey(fmt.Sprintf("RT%s", args[1].Value.(string)))}
					return nil
				},
			}},
		},
	}},
	"RTString": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTLength": {FunLiteral, NativeMethod{
				Args: []string{"str"},
				Body: func(vm *VM, args []*Literal) error {
					if len(args) < 1 {
						return nil
					}

					acc := args[0]

					if acc.LiteralType != StringLiteral {
						return nil
					}

					vm.ReturnValue = &Literal{IntLiteral, len(acc.Value.(string))}
					return nil
				},
			}},
			"RTAt": {FunLiteral, NativeMethod{
				Args: []string{"str", "idx"},
				Body: func(vm *VM, args []*Literal) error {
					if len(args) < 2 {
						return nil
					}

					acc := args[0]

					if acc.LiteralType != StringLiteral {
						return nil
					}

					if args[1].LiteralType != IntLiteral {
						return nil
					}

					vm.ReturnValue = &Literal{StringLiteral, string(acc.Value.(string)[args[1].Value.(int)])}
					return nil
				},
			}},
			"RTSubstring": {FunLiteral, NativeMethod{
				Args: []string{"str", "start", "end"},
				Body: func(vm *VM, args []*Literal) error {
					if len(args) < 2 {
						return nil
					}

					acc := args[0]

					if acc.LiteralType != StringLiteral {
						return nil
					}

					if args[1].LiteralType != IntLiteral {
						return nil
					}

					if len(args) > 2 && args[2].LiteralType != IntLiteral {
						return nil
					}

					if len(args) == 2 {
						vm.ReturnValue = &Literal{StringLiteral, acc.Value.(string)[args[1].Value.(int):]}
						return nil
					}

					vm.ReturnValue = &Literal{StringLiteral, acc.Value.(string)[args[1].Value.(int):args[2].Value.(int)]}
					return nil

				},
			}},
		},
	}},
	"RTInt": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTParse": {FunLiteral, NativeMethod{
				Args: []string{"str"},
				Body: func(vm *VM, args []*Literal) error {
					if len(args) < 1 {
						return nil
					}

					acc := args[0]

					if acc.LiteralType != StringLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					val, err := strconv.Atoi(acc.Value.(string))

					if err != nil {
						return err
					}

					vm.ReturnValue = &Literal{LiteralType: IntLiteral, Value: val}
					return nil
				},
			}},
		},
	}},
	"RTOption": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTNone": {ParsedObjLiteral, &PartsSpecialObject{
				Internal: &PartsObject{},
				Hash:     "Option.None",
			}},
			"RTSome": {FunLiteral, NativeMethod{
				Args: []string{"val"},
				Body: func(vm *VM, args []*Literal) error {

					vm.ReturnValue = &Literal{ParsedObjLiteral, &PartsSpecialObject{
						Internal: &PartsObject{Entries: map[string]*Literal{"RTValue": args[0]}},
						Hash:     "Option.Some",
					}}

					return nil
				},
			}},
			"RTIsSome": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) error {
					acc := args[0]

					if acc.LiteralType != ParsedObjLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					th := acc.Value.(PartsIndexable).TypeHash()

					if th == "Option.Some" {
						vm.ReturnValue = &Literal{BoolLiteral, true}
						return nil
					}

					vm.ReturnValue = &Literal{BoolLiteral, false}
					return nil
				},
			}},
			"RTIsNone": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) error {
					acc := args[0]

					if acc.LiteralType != ParsedObjLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					th := acc.Value.(PartsIndexable).TypeHash()

					if th == "Option.None" {
						vm.ReturnValue = &Literal{BoolLiteral, true}
						return nil
					}

					vm.ReturnValue = &Literal{BoolLiteral, false}
					return nil
				},
			}},
			"RTIsOption": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) error {
					acc := args[0]

					if acc.LiteralType != ParsedObjLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					th := acc.Value.(PartsIndexable).TypeHash()

					if th == "Option.None" || th == "Option.Some" {
						vm.ReturnValue = &Literal{BoolLiteral, true}
						return nil
					}

					vm.ReturnValue = &Literal{BoolLiteral, false}
					return nil
				},
			}},
		},
	}},
	"RTResult": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTError": {FunLiteral, NativeMethod{
				Args: []string{"val"},
				Body: func(vm *VM, args []*Literal) error {
					vm.ReturnValue = &Literal{ParsedObjLiteral, &PartsSpecialObject{
						Internal: &PartsObject{Entries: map[string]*Literal{"RTValue": args[0]}},
						Hash:     "Result.Error",
					}}

					return nil
				},
			}},
			"RTOk": {FunLiteral, NativeMethod{
				Args: []string{"val"},
				Body: func(vm *VM, args []*Literal) error {

					vm.ReturnValue = &Literal{ParsedObjLiteral, &PartsSpecialObject{
						Internal: &PartsObject{Entries: map[string]*Literal{"RTValue": args[0]}},
						Hash:     "Result.Ok",
					}}

					return nil
				},
			}},
			"RTIsOk": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) error {
					acc := args[0]

					if acc.LiteralType != ParsedObjLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					th := acc.Value.(PartsIndexable).TypeHash()

					if th == "Result.Ok" {
						vm.ReturnValue = &Literal{BoolLiteral, true}
						return nil
					}

					vm.ReturnValue = &Literal{BoolLiteral, false}
					return nil
				},
			}},
			"RTIsResult": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) error {
					acc := args[0]

					if acc.LiteralType != ParsedObjLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					th := acc.Value.(PartsIndexable).TypeHash()

					if th == "Result.Ok" || th == "Result.Error" {
						vm.ReturnValue = &Literal{BoolLiteral, true}
						return nil
					}

					vm.ReturnValue = &Literal{BoolLiteral, false}
					return nil
				},
			}},
			"RTIsError": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) error {
					acc := args[0]

					if acc.LiteralType != ParsedObjLiteral {
						vm.ReturnValue = &Literal{BoolLiteral, false}
						return nil
					}

					th := acc.Value.(PartsIndexable).TypeHash()

					if th == "Result.Error" {
						vm.ReturnValue = &Literal{BoolLiteral, true}
						return nil
					}

					vm.ReturnValue = &Literal{BoolLiteral, false}
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

type PartsSpecialObject struct {
	Internal *PartsObject
	Hash     string
}

func (pso PartsSpecialObject) Get(key *Literal) *Literal {
	return pso.Internal.Get(key)
}

func (pso PartsSpecialObject) Set(key *Literal, value *Literal) *Literal {
	return pso.Internal.Set(key, value)
}

func (pso PartsSpecialObject) Has(key *Literal) bool {
	return pso.Internal.Has(key)
}

func (pso PartsSpecialObject) Length() int {
	return pso.Internal.Length()
}

func (pso PartsSpecialObject) GetAll() map[string]*Literal {
	return pso.Internal.GetAll()
}

func (pso PartsSpecialObject) GetByKey(key string) *Literal {
	return pso.Internal.GetByKey(key)
}

func (pso PartsSpecialObject) SetByKey(key string, value *Literal) *Literal {
	return pso.Internal.SetByKey(key, value)
}

func (pso PartsSpecialObject) HasByKey(key string) bool {
	return pso.Internal.HasByKey(key)
}

func (pso PartsSpecialObject) TypeHash() string {
	return pso.Hash
}
