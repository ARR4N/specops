package specops

//
// GENERATED CODE - DO NOT EDIT
//

import (
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/arr4n/specops/types"
)

// Aliases of all regular vm.OpCode constants that don't have "special" replacements.
const (
	STOP = types.OpCode(vm.STOP)
	ADD = types.OpCode(vm.ADD)
	MUL = types.OpCode(vm.MUL)
	SUB = types.OpCode(vm.SUB)
	DIV = types.OpCode(vm.DIV)
	SDIV = types.OpCode(vm.SDIV)
	MOD = types.OpCode(vm.MOD)
	SMOD = types.OpCode(vm.SMOD)
	ADDMOD = types.OpCode(vm.ADDMOD)
	MULMOD = types.OpCode(vm.MULMOD)
	EXP = types.OpCode(vm.EXP)
	SIGNEXTEND = types.OpCode(vm.SIGNEXTEND)
	LT = types.OpCode(vm.LT)
	GT = types.OpCode(vm.GT)
	SLT = types.OpCode(vm.SLT)
	SGT = types.OpCode(vm.SGT)
	EQ = types.OpCode(vm.EQ)
	ISZERO = types.OpCode(vm.ISZERO)
	AND = types.OpCode(vm.AND)
	OR = types.OpCode(vm.OR)
	XOR = types.OpCode(vm.XOR)
	NOT = types.OpCode(vm.NOT)
	BYTE = types.OpCode(vm.BYTE)
	SHL = types.OpCode(vm.SHL)
	SHR = types.OpCode(vm.SHR)
	SAR = types.OpCode(vm.SAR)
	KECCAK256 = types.OpCode(vm.KECCAK256)
	ADDRESS = types.OpCode(vm.ADDRESS)
	BALANCE = types.OpCode(vm.BALANCE)
	ORIGIN = types.OpCode(vm.ORIGIN)
	CALLER = types.OpCode(vm.CALLER)
	CALLVALUE = types.OpCode(vm.CALLVALUE)
	CALLDATALOAD = types.OpCode(vm.CALLDATALOAD)
	CALLDATASIZE = types.OpCode(vm.CALLDATASIZE)
	CALLDATACOPY = types.OpCode(vm.CALLDATACOPY)
	CODESIZE = types.OpCode(vm.CODESIZE)
	CODECOPY = types.OpCode(vm.CODECOPY)
	GASPRICE = types.OpCode(vm.GASPRICE)
	EXTCODESIZE = types.OpCode(vm.EXTCODESIZE)
	EXTCODECOPY = types.OpCode(vm.EXTCODECOPY)
	RETURNDATASIZE = types.OpCode(vm.RETURNDATASIZE)
	RETURNDATACOPY = types.OpCode(vm.RETURNDATACOPY)
	EXTCODEHASH = types.OpCode(vm.EXTCODEHASH)
	BLOCKHASH = types.OpCode(vm.BLOCKHASH)
	COINBASE = types.OpCode(vm.COINBASE)
	TIMESTAMP = types.OpCode(vm.TIMESTAMP)
	NUMBER = types.OpCode(vm.NUMBER)
	DIFFICULTY = types.OpCode(vm.DIFFICULTY)
	GASLIMIT = types.OpCode(vm.GASLIMIT)
	CHAINID = types.OpCode(vm.CHAINID)
	SELFBALANCE = types.OpCode(vm.SELFBALANCE)
	BASEFEE = types.OpCode(vm.BASEFEE)
	BLOBHASH = types.OpCode(vm.BLOBHASH)
	BLOBBASEFEE = types.OpCode(vm.BLOBBASEFEE)
	POP = types.OpCode(vm.POP)
	MLOAD = types.OpCode(vm.MLOAD)
	MSTORE = types.OpCode(vm.MSTORE)
	MSTORE8 = types.OpCode(vm.MSTORE8)
	SLOAD = types.OpCode(vm.SLOAD)
	SSTORE = types.OpCode(vm.SSTORE)
	JUMP = types.OpCode(vm.JUMP)
	JUMPI = types.OpCode(vm.JUMPI)
	PC = types.OpCode(vm.PC)
	MSIZE = types.OpCode(vm.MSIZE)
	GAS = types.OpCode(vm.GAS)
	TLOAD = types.OpCode(vm.TLOAD)
	TSTORE = types.OpCode(vm.TSTORE)
	MCOPY = types.OpCode(vm.MCOPY)
	PUSH0 = types.OpCode(vm.PUSH0)
	DUP1 = types.OpCode(vm.DUP1)
	DUP2 = types.OpCode(vm.DUP2)
	DUP3 = types.OpCode(vm.DUP3)
	DUP4 = types.OpCode(vm.DUP4)
	DUP5 = types.OpCode(vm.DUP5)
	DUP6 = types.OpCode(vm.DUP6)
	DUP7 = types.OpCode(vm.DUP7)
	DUP8 = types.OpCode(vm.DUP8)
	DUP9 = types.OpCode(vm.DUP9)
	DUP10 = types.OpCode(vm.DUP10)
	DUP11 = types.OpCode(vm.DUP11)
	DUP12 = types.OpCode(vm.DUP12)
	DUP13 = types.OpCode(vm.DUP13)
	DUP14 = types.OpCode(vm.DUP14)
	DUP15 = types.OpCode(vm.DUP15)
	DUP16 = types.OpCode(vm.DUP16)
	SWAP1 = types.OpCode(vm.SWAP1)
	SWAP2 = types.OpCode(vm.SWAP2)
	SWAP3 = types.OpCode(vm.SWAP3)
	SWAP4 = types.OpCode(vm.SWAP4)
	SWAP5 = types.OpCode(vm.SWAP5)
	SWAP6 = types.OpCode(vm.SWAP6)
	SWAP7 = types.OpCode(vm.SWAP7)
	SWAP8 = types.OpCode(vm.SWAP8)
	SWAP9 = types.OpCode(vm.SWAP9)
	SWAP10 = types.OpCode(vm.SWAP10)
	SWAP11 = types.OpCode(vm.SWAP11)
	SWAP12 = types.OpCode(vm.SWAP12)
	SWAP13 = types.OpCode(vm.SWAP13)
	SWAP14 = types.OpCode(vm.SWAP14)
	SWAP15 = types.OpCode(vm.SWAP15)
	SWAP16 = types.OpCode(vm.SWAP16)
	LOG0 = types.OpCode(vm.LOG0)
	LOG1 = types.OpCode(vm.LOG1)
	LOG2 = types.OpCode(vm.LOG2)
	LOG3 = types.OpCode(vm.LOG3)
	LOG4 = types.OpCode(vm.LOG4)
	CREATE = types.OpCode(vm.CREATE)
	CALL = types.OpCode(vm.CALL)
	CALLCODE = types.OpCode(vm.CALLCODE)
	RETURN = types.OpCode(vm.RETURN)
	DELEGATECALL = types.OpCode(vm.DELEGATECALL)
	CREATE2 = types.OpCode(vm.CREATE2)
	STATICCALL = types.OpCode(vm.STATICCALL)
	REVERT = types.OpCode(vm.REVERT)
	INVALID = types.OpCode(vm.INVALID)
	SELFDESTRUCT = types.OpCode(vm.SELFDESTRUCT)
)

// stackDeltas maps all valid vm.OpCode values to the number of values they
// pop and then push from/to the stack.
//
// Although DUPs technically only push a single value and SWAPs none, they are
// recorded as popping and pushing one more than they actually do as this
// implies a minimum stack depth to begin with but with the same effective
// change.
var stackDeltas = map[vm.OpCode]stackDelta{
	vm.STOP: {pop: 0, push: 0},
	vm.ADD: {pop: 2, push: 1},
	vm.MUL: {pop: 2, push: 1},
	vm.SUB: {pop: 2, push: 1},
	vm.DIV: {pop: 2, push: 1},
	vm.SDIV: {pop: 2, push: 1},
	vm.MOD: {pop: 2, push: 1},
	vm.SMOD: {pop: 2, push: 1},
	vm.ADDMOD: {pop: 3, push: 1},
	vm.MULMOD: {pop: 3, push: 1},
	vm.EXP: {pop: 2, push: 1},
	vm.SIGNEXTEND: {pop: 2, push: 1},
	vm.LT: {pop: 2, push: 1},
	vm.GT: {pop: 2, push: 1},
	vm.SLT: {pop: 2, push: 1},
	vm.SGT: {pop: 2, push: 1},
	vm.EQ: {pop: 2, push: 1},
	vm.ISZERO: {pop: 1, push: 1},
	vm.AND: {pop: 2, push: 1},
	vm.OR: {pop: 2, push: 1},
	vm.XOR: {pop: 2, push: 1},
	vm.NOT: {pop: 1, push: 1},
	vm.BYTE: {pop: 2, push: 1},
	vm.SHL: {pop: 2, push: 1},
	vm.SHR: {pop: 2, push: 1},
	vm.SAR: {pop: 2, push: 1},
	vm.KECCAK256: {pop: 2, push: 1},
	vm.ADDRESS: {pop: 0, push: 1},
	vm.BALANCE: {pop: 1, push: 1},
	vm.ORIGIN: {pop: 0, push: 1},
	vm.CALLER: {pop: 0, push: 1},
	vm.CALLVALUE: {pop: 0, push: 1},
	vm.CALLDATALOAD: {pop: 1, push: 1},
	vm.CALLDATASIZE: {pop: 0, push: 1},
	vm.CALLDATACOPY: {pop: 3, push: 0},
	vm.CODESIZE: {pop: 0, push: 1},
	vm.CODECOPY: {pop: 3, push: 0},
	vm.GASPRICE: {pop: 0, push: 1},
	vm.EXTCODESIZE: {pop: 1, push: 1},
	vm.EXTCODECOPY: {pop: 4, push: 0},
	vm.RETURNDATASIZE: {pop: 0, push: 1},
	vm.RETURNDATACOPY: {pop: 3, push: 0},
	vm.EXTCODEHASH: {pop: 1, push: 1},
	vm.BLOCKHASH: {pop: 1, push: 1},
	vm.COINBASE: {pop: 0, push: 1},
	vm.TIMESTAMP: {pop: 0, push: 1},
	vm.NUMBER: {pop: 0, push: 1},
	vm.DIFFICULTY: {pop: 0, push: 1},
	vm.GASLIMIT: {pop: 0, push: 1},
	vm.CHAINID: {pop: 0, push: 1},
	vm.SELFBALANCE: {pop: 0, push: 1},
	vm.BASEFEE: {pop: 0, push: 1},
	vm.BLOBHASH: {pop: 1, push: 1},
	vm.BLOBBASEFEE: {pop: 0, push: 1},
	vm.POP: {pop: 1, push: 0},
	vm.MLOAD: {pop: 1, push: 1},
	vm.MSTORE: {pop: 2, push: 0},
	vm.MSTORE8: {pop: 2, push: 0},
	vm.SLOAD: {pop: 1, push: 1},
	vm.SSTORE: {pop: 2, push: 0},
	vm.JUMP: {pop: 1, push: 0},
	vm.JUMPI: {pop: 2, push: 0},
	vm.PC: {pop: 0, push: 1},
	vm.MSIZE: {pop: 0, push: 1},
	vm.GAS: {pop: 0, push: 1},
	vm.JUMPDEST: {pop: 0, push: 0},
	vm.TLOAD: {pop: 1, push: 1},
	vm.TSTORE: {pop: 2, push: 0},
	vm.MCOPY: {pop: 3, push: 0},
	vm.PUSH0: {pop: 0, push: 1},
	vm.PUSH1: {pop: 0, push: 1},
	vm.PUSH2: {pop: 0, push: 1},
	vm.PUSH3: {pop: 0, push: 1},
	vm.PUSH4: {pop: 0, push: 1},
	vm.PUSH5: {pop: 0, push: 1},
	vm.PUSH6: {pop: 0, push: 1},
	vm.PUSH7: {pop: 0, push: 1},
	vm.PUSH8: {pop: 0, push: 1},
	vm.PUSH9: {pop: 0, push: 1},
	vm.PUSH10: {pop: 0, push: 1},
	vm.PUSH11: {pop: 0, push: 1},
	vm.PUSH12: {pop: 0, push: 1},
	vm.PUSH13: {pop: 0, push: 1},
	vm.PUSH14: {pop: 0, push: 1},
	vm.PUSH15: {pop: 0, push: 1},
	vm.PUSH16: {pop: 0, push: 1},
	vm.PUSH17: {pop: 0, push: 1},
	vm.PUSH18: {pop: 0, push: 1},
	vm.PUSH19: {pop: 0, push: 1},
	vm.PUSH20: {pop: 0, push: 1},
	vm.PUSH21: {pop: 0, push: 1},
	vm.PUSH22: {pop: 0, push: 1},
	vm.PUSH23: {pop: 0, push: 1},
	vm.PUSH24: {pop: 0, push: 1},
	vm.PUSH25: {pop: 0, push: 1},
	vm.PUSH26: {pop: 0, push: 1},
	vm.PUSH27: {pop: 0, push: 1},
	vm.PUSH28: {pop: 0, push: 1},
	vm.PUSH29: {pop: 0, push: 1},
	vm.PUSH30: {pop: 0, push: 1},
	vm.PUSH31: {pop: 0, push: 1},
	vm.PUSH32: {pop: 0, push: 1},
	vm.DUP1: {pop: 1, push: 2},
	vm.DUP2: {pop: 1, push: 2},
	vm.DUP3: {pop: 1, push: 2},
	vm.DUP4: {pop: 1, push: 2},
	vm.DUP5: {pop: 1, push: 2},
	vm.DUP6: {pop: 1, push: 2},
	vm.DUP7: {pop: 1, push: 2},
	vm.DUP8: {pop: 1, push: 2},
	vm.DUP9: {pop: 1, push: 2},
	vm.DUP10: {pop: 1, push: 2},
	vm.DUP11: {pop: 1, push: 2},
	vm.DUP12: {pop: 1, push: 2},
	vm.DUP13: {pop: 1, push: 2},
	vm.DUP14: {pop: 1, push: 2},
	vm.DUP15: {pop: 1, push: 2},
	vm.DUP16: {pop: 1, push: 2},
	vm.SWAP1: {pop: 1, push: 1},
	vm.SWAP2: {pop: 1, push: 1},
	vm.SWAP3: {pop: 1, push: 1},
	vm.SWAP4: {pop: 1, push: 1},
	vm.SWAP5: {pop: 1, push: 1},
	vm.SWAP6: {pop: 1, push: 1},
	vm.SWAP7: {pop: 1, push: 1},
	vm.SWAP8: {pop: 1, push: 1},
	vm.SWAP9: {pop: 1, push: 1},
	vm.SWAP10: {pop: 1, push: 1},
	vm.SWAP11: {pop: 1, push: 1},
	vm.SWAP12: {pop: 1, push: 1},
	vm.SWAP13: {pop: 1, push: 1},
	vm.SWAP14: {pop: 1, push: 1},
	vm.SWAP15: {pop: 1, push: 1},
	vm.SWAP16: {pop: 1, push: 1},
	vm.LOG0: {pop: 2, push: 0},
	vm.LOG1: {pop: 3, push: 0},
	vm.LOG2: {pop: 4, push: 0},
	vm.LOG3: {pop: 5, push: 0},
	vm.LOG4: {pop: 6, push: 0},
	vm.CREATE: {pop: 3, push: 1},
	vm.CALL: {pop: 7, push: 1},
	vm.CALLCODE: {pop: 7, push: 1},
	vm.RETURN: {pop: 2, push: 0},
	vm.DELEGATECALL: {pop: 6, push: 1},
	vm.CREATE2: {pop: 4, push: 1},
	vm.STATICCALL: {pop: 6, push: 1},
	vm.REVERT: {pop: 2, push: 0},
	vm.INVALID: {pop: 0, push: 0},
	vm.SELFDESTRUCT: {pop: 1, push: 0},
}
