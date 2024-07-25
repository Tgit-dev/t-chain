package staking

import (
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/helper/keccak"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/0xPolygon/polygon-edge/validators"
)

var (
	MinValidatorCount = uint64(1)
	MaxValidatorCount = common.MaxSafeJSInt
)

// getAddressMapping returns the key for the SC storage mapping (address => something)
//
// More information:
// https://docs.soliditylang.org/en/latest/internals/layout_in_storage.html
func getAddressMapping(address types.Address, slot int64) []byte {
	bigSlot := big.NewInt(slot)

	finalSlice := append(
		common.PadLeftOrTrim(address.Bytes(), 32),
		common.PadLeftOrTrim(bigSlot.Bytes(), 32)...,
	)

	return keccak.Keccak256(nil, finalSlice)
}

// getIndexWithOffset is a helper method for adding an offset to the already found keccak hash
func getIndexWithOffset(keccakHash []byte, offset uint64) []byte {
	bigOffset := big.NewInt(int64(offset))
	bigKeccak := big.NewInt(0).SetBytes(keccakHash)

	bigKeccak.Add(bigKeccak, bigOffset)

	return bigKeccak.Bytes()
}

// getStorageIndexes is a helper function for getting the correct indexes
// of the storage slots which need to be modified during bootstrap.
//
// It is SC dependant, and based on the SC located at:
// https://github.com/0xPolygon/staking-contracts/
func getStorageIndexes(validator validators.Validator, index int) *StorageIndexes {
	storageIndexes := &StorageIndexes{}
	address := validator.Addr()

	// Get the indexes for the mappings
	// The index for the mapping is retrieved with:
	// keccak(address . slot)
	// . stands for concatenation (basically appending the bytes)
	storageIndexes.AddressToIsValidatorIndex = getAddressMapping(
		address,
		addressToIsValidatorSlot,
	)

	storageIndexes.AddressToStakedAmountIndex = getAddressMapping(
		address,
		addressToStakedAmountSlot,
	)

	storageIndexes.AddressToValidatorIndexIndex = getAddressMapping(
		address,
		addressToValidatorIndexSlot,
	)

	storageIndexes.ValidatorBLSPublicKeyIndex = getAddressMapping(
		address,
		addressToBLSPublicKeySlot,
	)

	// Index for array types is calculated as keccak(slot) + index
	// The slot for the dynamic arrays that's put in the keccak needs to be in hex form (padded 64 chars)
	storageIndexes.ValidatorsIndex = getIndexWithOffset(
		keccak.Keccak256(nil, common.PadLeftOrTrim(big.NewInt(validatorsSlot).Bytes(), 32)),
		uint64(index),
	)

	return storageIndexes
}

// setBytesToStorage sets bytes data into storage map from specified base index
func setBytesToStorage(
	storageMap map[types.Hash]types.Hash,
	baseIndexBytes []byte,
	data []byte,
) {
	dataLen := len(data)
	baseIndex := types.BytesToHash(baseIndexBytes)

	if dataLen <= 31 {
		bytes := types.Hash{}

		copy(bytes[:len(data)], data)

		// Set 2*Size at the first byte
		bytes[len(bytes)-1] = byte(dataLen * 2)

		storageMap[baseIndex] = bytes

		return
	}

	// Set size at the base index
	baseSlot := types.Hash{}
	baseSlot[31] = byte(2*dataLen + 1)
	storageMap[baseIndex] = baseSlot

	zeroIndex := keccak.Keccak256(nil, baseIndexBytes)
	numBytesInSlot := 256 / 8

	for i := 0; i < dataLen; i++ {
		offset := i / numBytesInSlot

		slotIndex := types.BytesToHash(getIndexWithOffset(zeroIndex, uint64(offset)))
		byteIndex := i % numBytesInSlot

		slot := storageMap[slotIndex]
		slot[byteIndex] = data[i]

		storageMap[slotIndex] = slot
	}
}

// PredeployParams contains the values used to predeploy the PoS staking contract
type PredeployParams struct {
	MinValidatorCount uint64
	MaxValidatorCount uint64
}

// StorageIndexes is a wrapper for different storage indexes that
// need to be modified
type StorageIndexes struct {
	ValidatorsIndex              []byte // []address
	ValidatorBLSPublicKeyIndex   []byte // mapping(address => byte[])
	AddressToIsValidatorIndex    []byte // mapping(address => bool)
	AddressToStakedAmountIndex   []byte // mapping(address => uint256)
	AddressToValidatorIndexIndex []byte // mapping(address => uint256)
}

// Slot definitions for SC storage
var (
	validatorsSlot              = int64(0) // Slot 0
	addressToIsValidatorSlot    = int64(1) // Slot 1
	addressToStakedAmountSlot   = int64(2) // Slot 2
	addressToValidatorIndexSlot = int64(3) // Slot 3
	stakedAmountSlot            = int64(4) // Slot 4
	minNumValidatorSlot         = int64(5) // Slot 5
	maxNumValidatorSlot         = int64(6) // Slot 6
	addressToBLSPublicKeySlot   = int64(7) // Slot 7
)

const (
	DefaultStakedBalance = "0x0" // 0 ETH
	//nolint: lll
	StakingSCBytecode = "0x6080604052600436106101185760003560e01c80637a6eea37116100a0578063d94c111b11610064578063d94c111b1461040a578063e387a7ed14610433578063e804fbf61461045e578063f90ecacc14610489578063facd743b146104c657610186565b80637a6eea37146103215780637dceceb81461034c578063af6da36e14610389578063c795c077146103b4578063ca1e7819146103df57610186565b8063373d6132116100e7578063373d6132146102595780633a4b66f1146102845780633c561f041461028e57806351a9ab32146102b9578063714ff425146102f657610186565b806302b751991461018b578063065ae171146101c85780632367f6b5146102055780632def66201461024257610186565b366101865761013c3373ffffffffffffffffffffffffffffffffffffffff16610503565b1561017c576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610173906117ec565b60405180910390fd5b610184610526565b005b600080fd5b34801561019757600080fd5b506101b260048036038101906101ad91906113e2565b6105fd565b6040516101bf9190611847565b60405180910390f35b3480156101d457600080fd5b506101ef60048036038101906101ea91906113e2565b610615565b6040516101fc919061174f565b60405180910390f35b34801561021157600080fd5b5061022c600480360381019061022791906113e2565b610635565b6040516102399190611847565b60405180910390f35b34801561024e57600080fd5b5061025761067e565b005b34801561026557600080fd5b5061026e610769565b60405161027b9190611847565b60405180910390f35b61028c610773565b005b34801561029a57600080fd5b506102a36107dc565b6040516102b0919061172d565b60405180910390f35b3480156102c557600080fd5b506102e060048036038101906102db91906113e2565b610982565b6040516102ed919061176a565b60405180910390f35b34801561030257600080fd5b5061030b610a22565b6040516103189190611847565b60405180910390f35b34801561032d57600080fd5b50610336610a2c565b604051610343919061182c565b60405180910390f35b34801561035857600080fd5b50610373600480360381019061036e91906113e2565b610a3a565b6040516103809190611847565b60405180910390f35b34801561039557600080fd5b5061039e610a52565b6040516103ab9190611847565b60405180910390f35b3480156103c057600080fd5b506103c9610a58565b6040516103d69190611847565b60405180910390f35b3480156103eb57600080fd5b506103f4610a5e565b604051610401919061170b565b60405180910390f35b34801561041657600080fd5b50610431600480360381019061042c919061140f565b610aec565b005b34801561043f57600080fd5b50610448610b91565b6040516104559190611847565b60405180910390f35b34801561046a57600080fd5b50610473610b97565b6040516104809190611847565b60405180910390f35b34801561049557600080fd5b506104b060048036038101906104ab9190611458565b610ba1565b6040516104bd91906116f0565b60405180910390f35b3480156104d257600080fd5b506104ed60048036038101906104e891906113e2565b610be0565b6040516104fa919061174f565b60405180910390f35b6000808273ffffffffffffffffffffffffffffffffffffffff163b119050919050565b34600460008282546105389190611968565b9250508190555034600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825461058e9190611968565b9250508190555061059e33610c36565b156105ad576105ac33610cb0565b5b3373ffffffffffffffffffffffffffffffffffffffff167f9e71bc8eea02a63969f509818f2dafb9254532904319f9dbda79b67bd34a5f3d346040516105f39190611847565b60405180910390a2565b60036020528060005260406000206000915090505481565b60016020528060005260406000206000915054906101000a900460ff1681565b6000600260008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b61069d3373ffffffffffffffffffffffffffffffffffffffff16610503565b156106dd576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016106d4906117ec565b60405180910390fd5b6000600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541161075f576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107569061178c565b60405180910390fd5b610767610dff565b565b6000600454905090565b6107923373ffffffffffffffffffffffffffffffffffffffff16610503565b156107d2576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107c9906117ec565b60405180910390fd5b6107da610526565b565b60606000808054905067ffffffffffffffff8111156107fe576107fd611c00565b5b60405190808252806020026020018201604052801561083157816020015b606081526020019060019003908161081c5790505b50905060005b60008054905081101561097a576007600080838154811061085b5761085a611bd1565b5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002080546108cb90611a98565b80601f01602080910402602001604051908101604052809291908181526020018280546108f790611a98565b80156109445780601f1061091957610100808354040283529160200191610944565b820191906000526020600020905b81548152906001019060200180831161092757829003601f168201915b505050505082828151811061095c5761095b611bd1565b5b6020026020010181905250808061097290611afb565b915050610837565b508091505090565b600760205280600052604060002060009150905080546109a190611a98565b80601f01602080910402602001604051908101604052809291908181526020018280546109cd90611a98565b8015610a1a5780601f106109ef57610100808354040283529160200191610a1a565b820191906000526020600020905b8154815290600101906020018083116109fd57829003601f168201915b505050505081565b6000600554905090565b69054b40b1f852bda0000081565b60026020528060005260406000206000915090505481565b60065481565b60055481565b60606000805480602002602001604051908101604052809291908181526020018280548015610ae257602002820191906000526020600020905b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610a98575b5050505050905090565b80600760003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000209080519060200190610b3f9291906112a5565b503373ffffffffffffffffffffffffffffffffffffffff167f472da4d064218fa97032725fbcff922201fa643fed0765b5ffe0ceef63d7b3dc82604051610b86919061176a565b60405180910390a250565b60045481565b6000600654905090565b60008181548110610bb157600080fd5b906000526020600020016000915054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff169050919050565b6000610c4182610f51565b158015610ca9575069054b40b1f852bda000006fffffffffffffffffffffffffffffffff16600260008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410155b9050919050565b60065460008054905010610cf9576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610cf0906117ac565b60405180910390fd5b60018060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff021916908315150217905550600080549050600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506000819080600181540180825580915050600190039060005260206000200160009091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b6000600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205490506000600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508060046000828254610e9a91906119be565b92505081905550610eaa33610f51565b15610eb957610eb833610fa7565b5b3373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f19350505050158015610eff573d6000803e3d6000fd5b503373ffffffffffffffffffffffffffffffffffffffff167f0f5bb82176feb1b5e747e28471aa92156a04d9f3ab9f45f28e2d704232b93f7582604051610f469190611847565b60405180910390a250565b6000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff169050919050565b60055460008054905011610ff0576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610fe79061180c565b60405180910390fd5b600080549050600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410611076576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161106d906117cc565b60405180910390fd5b6000600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050600060016000805490506110ce91906119be565b90508082146111bc5760008082815481106110ec576110eb611bd1565b5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050806000848154811061112e5761112d611bd1565b5b9060005260206000200160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555082600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550505b6000600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055506000600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550600080548061126b5761126a611ba2565b5b6001900381819060005260206000200160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690559055505050565b8280546112b190611a98565b90600052602060002090601f0160209004810192826112d3576000855561131a565b82601f106112ec57805160ff191683800117855561131a565b8280016001018555821561131a579182015b828111156113195782518255916020019190600101906112fe565b5b509050611327919061132b565b5090565b5b8082111561134457600081600090555060010161132c565b5090565b600061135b61135684611887565b611862565b90508281526020810184848401111561137757611376611c34565b5b611382848285611a56565b509392505050565b60008135905061139981611d6d565b92915050565b600082601f8301126113b4576113b3611c2f565b5b81356113c4848260208601611348565b91505092915050565b6000813590506113dc81611d84565b92915050565b6000602082840312156113f8576113f7611c3e565b5b60006114068482850161138a565b91505092915050565b60006020828403121561142557611424611c3e565b5b600082013567ffffffffffffffff81111561144357611442611c39565b5b61144f8482850161139f565b91505092915050565b60006020828403121561146e5761146d611c3e565b5b600061147c848285016113cd565b91505092915050565b600061149183836114b1565b60208301905092915050565b60006114a983836115b1565b905092915050565b6114ba816119f2565b82525050565b6114c9816119f2565b82525050565b60006114da826118d8565b6114e48185611913565b93506114ef836118b8565b8060005b838110156115205781516115078882611485565b9750611512836118f9565b9250506001810190506114f3565b5085935050505092915050565b6000611538826118e3565b6115428185611924565b935083602082028501611554856118c8565b8060005b858110156115905784840389528151611571858261149d565b945061157c83611906565b925060208a01995050600181019050611558565b50829750879550505050505092915050565b6115ab81611a04565b82525050565b60006115bc826118ee565b6115c68185611935565b93506115d6818560208601611a65565b6115df81611c43565b840191505092915050565b60006115f5826118ee565b6115ff8185611946565b935061160f818560208601611a65565b61161881611c43565b840191505092915050565b6000611630601d83611957565b915061163b82611c54565b602082019050919050565b6000611653602783611957565b915061165e82611c7d565b604082019050919050565b6000611676601283611957565b915061168182611ccc565b602082019050919050565b6000611699601a83611957565b91506116a482611cf5565b602082019050919050565b60006116bc604083611957565b91506116c782611d1e565b604082019050919050565b6116db81611a10565b82525050565b6116ea81611a4c565b82525050565b600060208201905061170560008301846114c0565b92915050565b6000602082019050818103600083015261172581846114cf565b905092915050565b60006020820190508181036000830152611747818461152d565b905092915050565b600060208201905061176460008301846115a2565b92915050565b6000602082019050818103600083015261178481846115ea565b905092915050565b600060208201905081810360008301526117a581611623565b9050919050565b600060208201905081810360008301526117c581611646565b9050919050565b600060208201905081810360008301526117e581611669565b9050919050565b600060208201905081810360008301526118058161168c565b9050919050565b60006020820190508181036000830152611825816116af565b9050919050565b600060208201905061184160008301846116d2565b92915050565b600060208201905061185c60008301846116e1565b92915050565b600061186c61187d565b90506118788282611aca565b919050565b6000604051905090565b600067ffffffffffffffff8211156118a2576118a1611c00565b5b6118ab82611c43565b9050602081019050919050565b6000819050602082019050919050565b6000819050602082019050919050565b600081519050919050565b600081519050919050565b600081519050919050565b6000602082019050919050565b6000602082019050919050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b600061197382611a4c565b915061197e83611a4c565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156119b3576119b2611b44565b5b828201905092915050565b60006119c982611a4c565b91506119d483611a4c565b9250828210156119e7576119e6611b44565b5b828203905092915050565b60006119fd82611a2c565b9050919050565b60008115159050919050565b60006fffffffffffffffffffffffffffffffff82169050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b82818337600083830152505050565b60005b83811015611a83578082015181840152602081019050611a68565b83811115611a92576000848401525b50505050565b60006002820490506001821680611ab057607f821691505b60208210811415611ac457611ac3611b73565b5b50919050565b611ad382611c43565b810181811067ffffffffffffffff82111715611af257611af1611c00565b5b80604052505050565b6000611b0682611a4c565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff821415611b3957611b38611b44565b5b600182019050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080fd5b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4f6e6c79207374616b65722063616e2063616c6c2066756e6374696f6e000000600082015250565b7f56616c696461746f72207365742068617320726561636865642066756c6c206360008201527f6170616369747900000000000000000000000000000000000000000000000000602082015250565b7f696e646578206f7574206f662072616e67650000000000000000000000000000600082015250565b7f4f6e6c7920454f412063616e2063616c6c2066756e6374696f6e000000000000600082015250565b7f56616c696461746f72732063616e2774206265206c657373207468616e20746860008201527f65206d696e696d756d2072657175697265642076616c696461746f72206e756d602082015250565b611d76816119f2565b8114611d8157600080fd5b50565b611d8d81611a4c565b8114611d9857600080fd5b5056fea26469706673582212203d21b6639ac0e2f7ad3b2028a115c8d88f41b534d866e20e10685a8450f2c5bf64736f6c63430008070033"

)

// PredeployStakingSC is a helper method for setting up the staking smart contract account,
// using the passed in validators as pre-staked validators
func PredeployStakingSC(
	vals validators.Validators,
	params PredeployParams,
) (*chain.GenesisAccount, error) {
	// Set the code for the staking smart contract
	// Code retrieved from https://github.com/0xPolygon/staking-contracts
	scHex, _ := hex.DecodeHex(StakingSCBytecode)
	stakingAccount := &chain.GenesisAccount{
		Code: scHex,
	}

	// Parse the default staked balance value into *big.Int
	val := DefaultStakedBalance
	bigDefaultStakedBalance, err := types.ParseUint256orHex(&val)

	if err != nil {
		return nil, fmt.Errorf("unable to generate DefaultStatkedBalance, %w", err)
	}

	// Generate the empty account storage map
	storageMap := make(map[types.Hash]types.Hash)
	bigTrueValue := big.NewInt(1)
	stakedAmount := big.NewInt(0)
	bigMinNumValidators := big.NewInt(int64(params.MinValidatorCount))
	bigMaxNumValidators := big.NewInt(int64(params.MaxValidatorCount))
	valsLen := big.NewInt(0)

	if vals != nil {
		valsLen = big.NewInt(int64(vals.Len()))

		for idx := 0; idx < vals.Len(); idx++ {
			validator := vals.At(uint64(idx))

			// Update the total staked amount
			stakedAmount = stakedAmount.Add(stakedAmount, bigDefaultStakedBalance)

			// Get the storage indexes
			storageIndexes := getStorageIndexes(validator, idx)

			// Set the value for the validators array
			storageMap[types.BytesToHash(storageIndexes.ValidatorsIndex)] =
				types.BytesToHash(
					validator.Addr().Bytes(),
				)

			if blsValidator, ok := validator.(*validators.BLSValidator); ok {
				setBytesToStorage(
					storageMap,
					storageIndexes.ValidatorBLSPublicKeyIndex,
					blsValidator.BLSPublicKey,
				)
			}

			// Set the value for the address -> validator array index mapping
			storageMap[types.BytesToHash(storageIndexes.AddressToIsValidatorIndex)] =
				types.BytesToHash(bigTrueValue.Bytes())

			// Set the value for the address -> staked amount mapping
			storageMap[types.BytesToHash(storageIndexes.AddressToStakedAmountIndex)] =
				types.StringToHash(hex.EncodeBig(bigDefaultStakedBalance))

			// Set the value for the address -> validator index mapping
			storageMap[types.BytesToHash(storageIndexes.AddressToValidatorIndexIndex)] =
				types.StringToHash(hex.EncodeUint64(uint64(idx)))
		}
	}

	// Set the value for the total staked amount
	storageMap[types.BytesToHash(big.NewInt(stakedAmountSlot).Bytes())] =
		types.BytesToHash(stakedAmount.Bytes())

	// Set the value for the size of the validators array
	storageMap[types.BytesToHash(big.NewInt(validatorsSlot).Bytes())] =
		types.BytesToHash(valsLen.Bytes())

	// Set the value for the minimum number of validators
	storageMap[types.BytesToHash(big.NewInt(minNumValidatorSlot).Bytes())] =
		types.BytesToHash(bigMinNumValidators.Bytes())

	// Set the value for the maximum number of validators
	storageMap[types.BytesToHash(big.NewInt(maxNumValidatorSlot).Bytes())] =
		types.BytesToHash(bigMaxNumValidators.Bytes())

	// Save the storage map
	stakingAccount.Storage = storageMap

	// Set the Staking SC balance to numValidators * defaultStakedBalance
	stakingAccount.Balance = stakedAmount

	return stakingAccount, nil
}
