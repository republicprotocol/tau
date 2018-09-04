# Instructions

## `push $value`

Push a value onto the stack.

## `rand`

Generate a random field element.

**Arguments**

None.

**Returns**

Pushes the random field element to the stack.

## `add`

Add two field elements together.

**Arguments**

1. Value to be added
2. Value to be added

**Returns**

1. Result of addition

## `mul`

Multiply two field elements together.

**Arguments**

1. Random number
2. Value to be multiplied
3. Value to be multiplied

**Returns**

1. Result of multiplication

## `dup`

Duplicate a value on the stack.

**Arguments**

1. Number of duplications
2. Value to be duplicated

**Returns**

1. Value to be duplicated, pushed multiple times

## `store`

Store a value in memory.

**Arguments**

1. Destination address in memory
2. Value to be stored

**Returns**

Nothing.

## `load`

Load a value from memory.

**Arguments**

1. Source address in memory

**Returns**

1. Value stored at address in memory (or zero)