# Declare (B_DECLARE)

Declares variable in the current scope.

## Structure:
Key followed by value.

## Example:

For literals:
- 2 - Reference, "x"
- 3 - Int, 10

Code:
`let x = 10`

Bytecode:
[B_DECLARE, B_LITERAL, 2, B_LITERAL, 3]

## Notable things

- Both the key and the value are expected to be non-null.
- Key can be of any type as long it's hashable

# Set (B_SET)

Set value of the variable in the current scope.

## Structure:
Key followed by value.

## Example (1):

For literals:
- 2 - Reference, "x"
- 3 - Reference, "y"
- 4 - Int, 10

Code:
`x.y = 10`

Bytecode:
[B_SET, B_DOT, B_LITERAL 2, B_LITERAL, 3, B_LITERAL, 4]

## Example (2):

For literals:
- 2 - Reference, "x"
- 3 - Int, 10

Code:
`x = 10`

Bytecode:
[B_SET, B_LITERAL, 2, B_LITERAL, 3]

## Notable things

- Both the key and the value are expected to be non-null.
- Key can be of any type as long it's hashable
- Set also works on values that are indexable

# Literal (B_LITERAL)

Reference to the literal at the specified index

## Structure:
Index encoded the following way:
- idx < 126 => value
- idx == 126 => two following bytes are index (uint16)
- idx == 127 => eight following bytes are index (uint64)

## Example:

For literals:
- 2 - Int, 10

Code:
`10`

Bytecode:
[B_LITERAL, 2]

# Return (B_RETURN)

Returns the value out of the current scope

## Structure:
Value

## Example:

For literals:
- 2 - Reference, "x"

Code:
`retrun x`

Bytecode:
[B_RETURN, B_LITERAL, 2]

## Notable things

- Return is used to return out of the scope, meaning it also returns out of if's `then` and `else` blocks
- When run outside of the scope (i.e. not in function body), this does not return.

# Raise (B_RAISE)

Raises value out of current scope

Raising in this context meaning returning error, i.e. returns Result.Error(val) type.

## Structure:
Value

## Example:

For literals:
- 2 - Reference, "x"

Code:
`raise x`

Bytecode:
[B_RAISE, B_LITERAL, 2]

## Notable things

- Behaves the same way as B_RETURN does.

# New scope (B_NEW_SCOPE)

Creates new scope for the VM environment

## Structure:
None

## Example:

For literals:
- 2 - Reference, "x"

Code:
`{return x}`

Bytecode:
[B_NEW_SCOPE, B_RETURN, B_LITERAL, 2, B_END_SCOPE]

# Exit scope (B_END_SCOPE)

Creates exits scope for the VM environment

## Structure:
None

## Example:

For literals:
- 2 - Reference, "x"

Code:
`{return x}`

Bytecode:
[B_NEW_SCOPE, B_RETURN, B_LITERAL, 2, B_END_SCOPE]

# Dot (B_DOT)

Accesses the value at the following key on the parent value

## Structure:
Key, Value

## Example:

For literals:
- 2 - Reference, "x"
- 3 - Reference, "y"

Code:
`x.y`

Bytecode:
[B_DOT, B_LITERAL, 2, B_LITERAL, 3]

# Function call (B_CALL)

Executes the function call operation

## Structure:
Value, arg count, args

## Example:

For literals:
- 2 - Reference, "x"
- 3 - Int, 0

Code:
`x(0)`

Bytecode:
[B_CALL, B_LITERAL, 2, 1, B_LITERAL, 3]

# Resolve (B_RESOLVE)

Resolves value

## Structure:
Value

## Example:

For literals:
- 2 - Reference, "val"
- 3 - Int, 10
- 4 - Reference, "x"

Code:
`let val = 10; x[y]`

Bytecode:
[
B_DECLARE B_LITERAL, 2, B_LITERAL, 3,

B_DOT, B_LITERAL, 2, B_RESOLVE, B_LITERAL, 3
]

## Notable things

- It's used to tell the VM the resolve the value instead of using it as it is, for examples, without that in the following example the dot operation would try to resolve `val` key of the object instead of the reference to 10.

# Conditional jump (B_COND_JUMP)

Performs the jump depending on the boolean value of the condition

## Structure:
Condition, `Then` length, Then block, `Else` length, Else block

## Example:

For literals:
- 2 - Reference, "x"
- 3 - Int, 0
- 4 - Int, 1

Code:
`if x { 0 } else { 1 }`

Bytecode:
[
B_COND_JUMP, B_LITERAL, 2, 4, B_NEW_SCOPE, B_LITERAL, 3, B_END_SCOPE, 4, B_NEW_SCOPE, B_LITERAL, 4, B_END_SCOPE
]

# Jump (B_JUMP)

Performs the jump

## Structure:
Offset

Coded the same way as in B_LITERAL

# Reverse jump (B_JUMP_REV)

Performs the jump backwards

## Structure:
Offset

Coded the same way as in B_LITERAL

# Binary operation (B_BIN_OP)

Performs the specified binary operation

## Structure:
Kind of operation, Left side, Right side

## Example:

For literals:
- 2 - Reference, "x"
- 3 - Int, 0

Code:
`x == 0`

Bytecode:
[ B_BIN_OP, B_OP_EQ, B_LITERAL, 2, B_LITERAL, 3 ]