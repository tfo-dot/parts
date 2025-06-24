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
		Body: func(vm *VM, args []*Literal) (*Literal, error) {
			fmt.Print(args[0].pretify())
			return nil, nil
		},
	}},
	"RTprintLn": {FunLiteral, NativeMethod{
		Args: []string{"arg"},
		Body: func(vm *VM, args []*Literal) (*Literal, error) {
			fmt.Println(args[0].pretify())
			return nil, nil
		},
	}},
	"RTreadLn": {FunLiteral, NativeMethod{
		Args: []string{},
		Body: func(vm *VM, args []*Literal) (*Literal, error) {
			reader := bufio.NewReader(os.Stdin)
			text, err := reader.ReadString('\n')

			if err != nil {
				return nil, errors.Join(errors.New("got error while reading os.Stdin"), err)
			}

			return &Literal{StringLiteral, text}, nil
		},
	}},
	"RTArray": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTHas": {FunLiteral, NativeMethod{
				Args: []string{"arr", "elt"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					acc := args[0]

					if acc.LiteralType != ParsedListLiteral {
						return &Literal{BoolLiteral, false}, nil
					}

					for _, val := range acc.Value.(PartsObject).Entries {
						if val, err := args[1].opEq(val); err != nil && val.Value.(bool) {
							return &Literal{BoolLiteral, true}, nil
						}
					}

					return &Literal{BoolLiteral, false}, nil
				},
			}},
			"RTAppendAll": {FunLiteral, NativeMethod{
				Args: []string{"arr", "other"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != ParsedListLiteral {
						return nil, errors.New("expected array type as a first argument to Array.AppendAll")
					}

					if args[1].LiteralType != ParsedListLiteral {
						println(args[1].LiteralType)
						return nil, errors.New("expected array type as a second argument to Array.AppendAll")
					}

					entries := args[1].Value.(PartsIndexable).GetAll()

					keys := make([]string, 0, len(entries))

					for k := range entries {
						keys = append(keys, k)
					}

					sort.Slice(keys, func(i, j int) bool {
						numStrI := strings.TrimPrefix(keys[i], "IT")
						numI, errI := strconv.Atoi(numStrI)

						if errI != nil {
							panic(fmt.Sprintf("could not parse number from key %s: %v\n", keys[i], errI))
						}

						numStrJ := strings.TrimPrefix(keys[j], "IT")
						numJ, errJ := strconv.Atoi(numStrJ)
						if errJ != nil {
							panic(fmt.Sprintf("could not parse number from key %s: %v\n", keys[j], errJ))
						}

						return numI < numJ
					})

					for _, key := range keys {
						args[0].Value.(PartsIndexable).SetByKey(fmt.Sprintf("IT%d", args[0].Value.(PartsIndexable).Length()), entries[key])
					}

					return args[0], nil
				},
			}},
			"RTSlice": {FunLiteral, NativeMethod{
				Args: []string{"arr", "start", "end"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != ParsedListLiteral {
						return nil, errors.New("expected array as a argument to Array.Slice")
					}

					if args[1].LiteralType != IntLiteral {
						return nil, errors.New("expected int as start index argument to Array.Slice")
					}

					if args[2].LiteralType != IntLiteral {
						return nil, errors.New("expected int as end index argument to Array.Slice")
					}

					entries := args[0].Value.(PartsIndexable).GetAll()

					keys := make([]string, 0, len(entries))

					for k := range entries {
						keys = append(keys, k)
					}

					sort.Slice(keys, func(i, j int) bool {
						numStrI := strings.TrimPrefix(keys[i], "IT")
						numI, errI := strconv.Atoi(numStrI)

						if errI != nil {
							panic(fmt.Sprintf("could not parse number from key %s: %v\n", keys[i], errI))
						}

						numStrJ := strings.TrimPrefix(keys[j], "IT")
						numJ, errJ := strconv.Atoi(numStrJ)
						if errJ != nil {
							panic(fmt.Sprintf("could not parse number from key %s: %v\n", keys[j], errJ))
						}

						return numI < numJ
					})

					slicedKeys := keys[args[1].Value.(int):args[2].Value.(int)]

					newArr := &PartsObject{Entries: make(map[string]*Literal)}

					for idx, elt := range slicedKeys {
						newArr.Set(&Literal{LiteralType: IntLiteral, Value: idx}, entries[elt])
					}

					return &Literal{LiteralType: ParsedListLiteral, Value: newArr}, nil
				},
			}},
			"RTLength": {FunLiteral, NativeMethod{
				Args: []string{"arr"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != ParsedListLiteral {
						return nil, errors.New("expected array as a argument to Array.Length")
					}

					return &Literal{LiteralType: IntLiteral, Value: args[0].Value.(PartsIndexable).Length()}, nil
				},
			}},
		},
	}},
	"RTObject": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTHas": {FunLiteral, NativeMethod{
				Args: []string{"obj", "elt"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != ParsedObjLiteral {
						return &Literal{BoolLiteral, false}, nil
					}

					if args[1].LiteralType != StringLiteral {
						return &Literal{BoolLiteral, false}, nil
					}

					casted, ok := args[0].Value.(PartsIndexable)

					if !ok {
						return nil, errors.New("obj is not parts PartsIndexable")
					}

					return &Literal{BoolLiteral, casted.HasByKey(fmt.Sprintf("RT%s", args[1].Value.(string)))}, nil
				},
			}},
			"RTKeys": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					casted, ok := args[0].Value.(PartsIndexable)

					if !ok {
						return nil, errors.New("obj is not parts PartsIndexable")
					}

					entries := casted.GetAll()

					keys := make([]string, 0, len(entries))

					for k := range entries {
						keys = append(keys, k)
					}

					newArr := &PartsObject{Entries: make(map[string]*Literal)}

					for idx, elt := range keys {
						newArr.Set(&Literal{LiteralType: IntLiteral, Value: idx}, &Literal{StringLiteral, elt[2:]})
					}

					return &Literal{LiteralType: ParsedListLiteral, Value: newArr}, nil
				},
			}},
			"RTAnyKey": {FunLiteral, NativeMethod{
				Args: []string{"obj", "key"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					casted, ok := args[0].Value.(PartsIndexable)

					if !ok {
						return nil, errors.New("obj is not parts PartsIndexable")
					}

					if args[1].LiteralType != StringLiteral {
						return nil, errors.New("expected string as an argument to Object.AnyKey")
					}

					query := args[1].Value.(string)

					entries := casted.GetAll()

					for key, val := range entries {
						if key[2:] == query {
							return val, nil
						}
					}

					return nil, nil
				},
			}},
		},
	}},
	"RTString": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTLength": {FunLiteral, NativeMethod{
				Args: []string{"str"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != StringLiteral {
						return nil, errors.New("expected string as a argument to String.Length")
					}

					return &Literal{IntLiteral, len(args[0].Value.(string))}, nil
				},
			}},
			"RTAt": {FunLiteral, NativeMethod{
				Args: []string{"str", "idx"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != StringLiteral {
						return nil, errors.New("expected string as a argument to String.At")
					}

					if args[1].LiteralType != IntLiteral {
						return nil, errors.New("expected int as index argument to String.At")
					}

					return &Literal{StringLiteral, string(args[0].Value.(string)[args[1].Value.(int)])}, nil
				},
			}},
			"RTSubstring": {FunLiteral, NativeMethod{
				Args: []string{"str", "start", "end"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != StringLiteral {
						return nil, errors.New("expected string as a argument to String.Substring")
					}

					if args[1].LiteralType != IntLiteral {
						return nil, errors.New("expected int as start index argument to String.Substring")
					}

					if args[2].LiteralType != IntLiteral {
						return nil, errors.New("expected int as end index argument to String.Substring")
					}

					return &Literal{StringLiteral, args[0].Value.(string)[args[1].Value.(int):args[2].Value.(int)]}, nil

				},
			}},
			"RTFrom": {FunLiteral, NativeMethod{
				Args: []string{"arg"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					return &Literal{StringLiteral, args[0].pretify()}, nil
				},
			}},
		},
	}},
	"RTInt": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTParse": {FunLiteral, NativeMethod{
				Args: []string{"str"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					acc := args[0]

					if acc.LiteralType != StringLiteral {
						return &Literal{BoolLiteral, false}, nil
					}

					val, err := strconv.Atoi(acc.Value.(string))

					if err != nil {
						return nil, err
					}

					return &Literal{LiteralType: IntLiteral, Value: val}, nil
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
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					return &Literal{ParsedObjLiteral, &PartsSpecialObject{
						Internal: &PartsObject{Entries: map[string]*Literal{"RTValue": args[0]}},
						Hash:     "Option.Some",
					}}, nil
				},
			}},
			"RTIsSome": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != ParsedObjLiteral {
						return &Literal{BoolLiteral, false}, nil
					}

					return &Literal{BoolLiteral, args[0].Value.(PartsIndexable).TypeHash() == "Option.Some"}, nil
				},
			}},
			"RTIsNone": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != ParsedObjLiteral {
						return &Literal{BoolLiteral, false}, nil
					}

					return &Literal{BoolLiteral, args[0].Value.(PartsIndexable).TypeHash() == "Option.None"}, nil
				},
			}},
			"RTIsOption": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					if args[0].LiteralType != ParsedObjLiteral {
						return &Literal{BoolLiteral, false}, nil
					}

					th := args[0].Value.(PartsIndexable).TypeHash()

					return &Literal{BoolLiteral, th == "Option.None" || th == "Option.Some"}, nil
				},
			}},
		},
	}},
	"RTResult": {ParsedObjLiteral, &PartsObject{
		Entries: map[string]*Literal{
			"RTError": {FunLiteral, NativeMethod{
				Args: []string{"val"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					return &Literal{ParsedObjLiteral, NewResultError(args[0])}, nil
				},
			}},
			"RTOk": {FunLiteral, NativeMethod{
				Args: []string{"val"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					return &Literal{ParsedObjLiteral, NewResultOK(args[0])}, nil
				},
			}},
			"RTIsOk": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					return &Literal{BoolLiteral, IsResultOK(args[0])}, nil
				},
			}},
			"RTIsResult": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					return &Literal{BoolLiteral, IsResult(args[0])}, nil
				},
			}},
			"RTIsError": {FunLiteral, NativeMethod{
				Args: []string{"obj"},
				Body: func(vm *VM, args []*Literal) (*Literal, error) {
					return &Literal{BoolLiteral, IsResultError(args[0])}, nil
				},
			}},
		},
	}},
}

type NativeMethod struct {
	Args []string
	Body func(vm *VM, args []*Literal) (*Literal, error)
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

	res, err := m.Body(vm, arguments)

	if err != nil {
		return err
	}

	if res != nil {
		vm.ReturnValue = res
		vm.ExitCode = ReturnCode
		vm.EarlyExit = true
	}

	return err
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

func IsResult(literal *Literal) bool {
	if literal.LiteralType != ParsedObjLiteral {
		return false
	}

	th := literal.Value.(PartsIndexable).TypeHash()

	return th == "Result.Error" || th == "Result.Ok"
}

func IsResultOK(literal *Literal) bool {
	if literal.LiteralType != ParsedObjLiteral {
		return false
	}

	return literal.Value.(PartsIndexable).TypeHash() == "Result.Ok"
}

func IsResultError(literal *Literal) bool {
	if literal.LiteralType != ParsedObjLiteral {
		return false
	}

	return literal.Value.(PartsIndexable).TypeHash() == "Result.Error"
}

func NewResultError(literal *Literal) *Literal {
	return &Literal{ParsedObjLiteral, &PartsSpecialObject{
		Internal: &PartsObject{Entries: map[string]*Literal{"RTValue": literal}},
		Hash:     "Result.Error",
	}}
}

func NewResultOK(literal *Literal) *Literal {
	return &Literal{ParsedObjLiteral, &PartsSpecialObject{
		Internal: &PartsObject{Entries: map[string]*Literal{"RTValue": literal}},
		Hash:     "Result.Ok",
	}}
}
