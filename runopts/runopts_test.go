package runopts_test

import (
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/solidifylabs/specops/revert"
	"github.com/solidifylabs/specops/runopts"

	. "github.com/solidifylabs/specops"
)

func randomAddresses(n int, seed []byte) []common.Address {
	keccak := crypto.NewKeccakState()
	keccak.Write(seed)

	addrs := make([]common.Address, n)
	buf := make([]byte, common.AddressLength)
	for i := range addrs {
		keccak.Read(buf) //nolint:errcheck // never returns an error
		copy(addrs[i][:], buf)
	}
	return addrs
}

func TestContractAddress(t *testing.T) {
	code := Code{
		Fn(MSTORE, PUSH0, ADDRESS),
		Fn(RETURN, PUSH(12), PUSH(20)),
	}

	addrs := randomAddresses(20, nil)

	for _, addr := range addrs {
		gotRes, err := code.Run(nil, runopts.ContractAddress(addr))
		if err != nil {
			t.Fatalf("%T.Run() error %v", code, err)
		}

		got := common.BytesToAddress(gotRes.Return())
		if want := addr; got != want {
			t.Errorf("contract deployed to address %v; want %v", got, want)
		}
	}
}

func TestFromAddress(t *testing.T) {
	code := Code{
		Fn(MSTORE, PUSH0, CALLER),
		Fn(RETURN, PUSH(12), PUSH(20)),
	}

	addrs := randomAddresses(20, nil)

	for _, addr := range addrs {
		gotRes, err := code.Run(nil, runopts.From(addr))
		if err != nil {
			t.Fatalf("%T.Run() error %v", code, err)
		}

		got := common.BytesToAddress(gotRes.Return())
		if want := addr; got != want {
			t.Errorf("contract called from address %v; want %v", got, want)
		}
	}
}

func TestValue(t *testing.T) {
	code := Code{
		Fn(MSTORE, PUSH0, CALLVALUE),
		Fn(RETURN, PUSH0, PUSH(0x20)),
	}

	keccak := crypto.NewKeccakState()
	buf := make([]byte, 16)
	vals := make([]uint256.Int, 20)
	for i := range vals {
		keccak.Read(buf) //nolint:errcheck // never returns an error
		vals[i].SetBytes(buf)
	}

	for _, val := range vals {
		gotRes, err := code.Run(nil, runopts.Value(val))
		if err != nil {
			t.Fatalf("%T.Run() error %v", code, err)
		}

		got := new(uint256.Int).SetBytes(gotRes.Return())
		if want := &val; !got.Eq(want) {
			t.Errorf("contract received value %v; want %v", got, want)
		}
	}
}

func TestErrorOnRevert(t *testing.T) {
	code := Code{INVALID}

	tests := []struct {
		name    string
		opts    []runopts.Option
		wantErr bool
	}{
		{
			name:    "without Options",
			wantErr: true,
		},
		{
			name:    "with NoErrorOnRevert Option",
			opts:    []runopts.Option{runopts.NoErrorOnRevert()},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := code.Run(nil, tt.opts...)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Errorf("%T.Run() got err %v; want err = %t", code, err, tt.wantErr)
			}
			if _, gotData := revert.Data(err); gotData != tt.wantErr {
				t.Errorf("revert.Data(error from %T.Run()) got %t; want %t", code, gotData, tt.wantErr)
			}

			// The ExecutionResult MUST always have the error.
			if typedErr, ok := res.Err.(*vm.ErrInvalidOpCode); !ok {
				t.Errorf("%T.Run() returned %T.Err = %T(%v); want %T", code, res, res.Err, res.Err, typedErr)
			}
		})
	}
}

func TestStorage(t *testing.T) {
	slot := common.Hash{'s', 'o', 'm', 'e', 'w', 'h', 'e', 'r', 'e'}
	const initVal = 42

	config := runopts.CaptureConfig()
	opts := []runopts.Option{
		config,
		runopts.Func(func(c *runopts.Configuration) error {
			c.StateDB.SetState(c.Contract.Address, slot, common.BigToHash(big.NewInt(initVal)))
			return nil
		}),
	}

	code := Code{
		Fn(SSTORE,
			PUSH(slot),
			Fn(ADD,
				PUSH(1),
				Fn(SLOAD, PUSH(slot)),
			),
		),
	}
	if _, err := code.Run(nil, opts...); err != nil {
		t.Fatalf("%T.Run() error %v", code, err)
	}

	cfg := config.Val
	got := cfg.StateDB.GetState(cfg.Contract.Address, slot).Big()
	want := big.NewInt(initVal + 1)
	if got.Cmp(want) != 0 {
		t.Errorf("got slot %v value = %v; want %v (initial value + 1)", slot, got, want)
	}
}

func TestGenesisAlloc(t *testing.T) {
	addr := common.Address{'a', 'd', 'd', 'r', 'e', 's', 's'}
	code := []byte{'c', 'o', 'd', 'e'}
	balance := big.NewInt(314159)

	slot := common.Hash{'s', 'l', 'o', 't'}
	data := common.Hash{'d', 'a', 't', 'a'}

	opt := runopts.GenesisAlloc(types.GenesisAlloc{
		addr: types.Account{
			Code:    code,
			Balance: balance,
		},
		runopts.DefaultContractAddress(): {
			Storage: map[common.Hash]common.Hash{
				slot: data,
			},
		},
	})

	c := Code{
		Fn(EXTCODEHASH, PUSH(addr)),
		Fn(BALANCE, PUSH(addr)),
		Fn(SLOAD, PUSH(slot)),
		INVALID, // TODO: add a mechanism for capturing stack/memory before they're cleared
	}
	want := make([]uint256.Int, 3)
	want[0].SetBytes32(crypto.Keccak256(code))
	want[1].SetFromBig(balance)
	want[2].SetBytes32(data[:])

	dbg, _, err := c.StartDebugging(nil, opt)
	if err != nil {
		t.Fatalf("%T.StartDebugging() error %v", code, err)
	}
	defer dbg.FastForward()

	var got []uint256.Int
	for !dbg.Done() {
		dbg.Step()
		if dbg.State().Op == vm.INVALID {
			got = dbg.State().Context.StackData()
			break
		}
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Stack diff (-want +got):\n%s", diff)
	}
}

func ExampleCaptured() {
	const (
		slot  = 42
		value = 314159
	)

	code := Code{
		Fn(SSTORE, PUSH(slot), PUSH(value)),
	}

	// All runopts.Captured[T] values are passed to Run() to be populated, after
	// which, their Val fields can be used.
	db := runopts.CaptureStateDB()
	if _, err := code.Run(nil, db); err != nil {
		log.Fatal(err)
	}

	got := db.Val.GetState(
		runopts.DefaultContractAddress(),
		common.BigToHash(big.NewInt(slot)),
	)
	fmt.Println(new(uint256.Int).SetBytes(got[:]))

	// Output: 314159
}
