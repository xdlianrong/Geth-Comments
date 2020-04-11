# EVM源码分析（合约间调用与预编译合约）

## 1 调用合约的具体流程

调用合约时对应的操作码

```go
CALLCODE: {
   execute:     opCallCode,
   constantGas: params.CallGasFrontier,
   dynamicGas:  gasCallCode,
   minStack:    minStack(7, 1),
   maxStack:    maxStack(7, 1),
   memorySize:  memoryCall,
   valid:       true,
   returns:     true,
},
```

下面是该操作码对应的执行函数

```go
func opCallCode(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
   // Pop gas. The actual gas is in interpreter.evm.callGasTemp.
   interpreter.intPool.put(stack.pop())
   gas := interpreter.evm.callGasTemp
   // Pop other call parameters.
   addr, value, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
   toAddr := common.BigToAddress(addr)
   value = math.U256(value)
   // Get arguments from the memory.
   args := memory.GetPtr(inOffset.Int64(), inSize.Int64())

   if value.Sign() != 0 {
      gas += params.CallStipend
   }
   ret, returnGas, err := interpreter.evm.CallCode(contract, toAddr, args, gas, value) // 向上调用evm方法CallCode，执行调用合约的函数
   if err != nil {
      stack.push(interpreter.intPool.getZero())
   } else {
      stack.push(interpreter.intPool.get().SetUint64(1))
   }
   if err == nil || err == errExecutionReverted {
      memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
   }
   contract.Gas += returnGas

   interpreter.intPool.put(addr, value, inOffset, inSize, retOffset, retSize)
   return ret, nil
}
```

下面是CallCode方法，该方法不像creat()和call()，该方法只能在指令中调用（向上面一样）

```go
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
   if evm.vmConfig.NoRecursion && evm.depth > 0 {
      return nil, gas, nil
   }

   // Fail if we're trying to execute above the call depth limit
   if evm.depth > int(params.CallCreateDepth) {
      return nil, gas, ErrDepth
   }
   // Fail if we're trying to transfer more than the available balance
   if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
      return nil, gas, ErrInsufficientBalance
   }

   var (
      snapshot = evm.StateDB.Snapshot()
      to       = AccountRef(caller.Address())
   )
   // Initialise a new contract and set the code that is to be used by the EVM.
   // The contract is a scoped environment for this execution context only.
   contract := NewContract(caller, to, value, gas)
   contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

   ret, err = run(evm, contract, input, false)
   // 同样这里要运行run方法，具体运行合约的具体内容，具体拿到runtime字段编译
   if err != nil {
      evm.StateDB.RevertToSnapshot(snapshot)
      if err != errExecutionReverted {
         contract.UseGas(contract.Gas)
      }
   }
   return ret, contract.Gas, err
}
```



## 3 调用合约示例

这里和文章（3）实例中的内容相似

这是一个合约间调用的简单示例

```java
pragma solidity ^0.4.11;

contract Foo {
}

contract FooFactory {
  address fooInstance;
  function makeNewFoo() {
    fooInstance = new Foo();
  }
}
```

编译后的字节码如下：

```
FooFactoryDeployCode
FooFactoryContractCode
  FooDeployCode
  FooContractCode
  FooAUXData
FooFactoryAUXData
```

1. 在该字节码中的binary主要先检测交易是否附带以太币，再把地址address的内容放入stateDB，再复制runtime和auxdata放入stateDB
2. runtime的内容为makeNewFoo函数，完成创建新合约Foo
3. autxdata不具体运行，合约代码的校验码和solc版本的数据

## 2 预编译合约

1. 预编译合约是以太坊内置的一些已经写好的合约，利用native Go来实现。这些合约是用来被我们正常写的合约调用的，不能独立于我们的合约自己运行。
2. 预编译合约是 EVM 中用于提供更复杂库函数(通常用于加密、散列等复杂操作)的一种折衷方法。因为计算量很大，所以预编译合约的内容并不是很容易用从操作码的方式实现（与evm存储空间容量的256bit，即32字节有关系）



回顾一下run函数，下面是run方法中的片段

```go
if contract.CodeAddr != nil {
      // 设置为适用于Frontier和Homestead阶段预编译以太坊集
      precompiles := PrecompiledContractsHomestead
      // 下面设置是拜占庭阶段的预编译以太坊集
      if evm.chainRules.IsByzantium {
         precompiles = PrecompiledContractsByzantium
      }
      // 下面是伊斯坦布尔阶段的预编译以太坊集
      if evm.chainRules.IsIstanbul {
         precompiles = PrecompiledContractsIstanbul
      }
      // 如果是上述阶段的特殊地址，RunPrecompiledContract调用其阶段对应的Run方法返回的字节码结果，并输出
      if p := precompiles[*contract.CodeAddr]; p != nil {
         // 在这里具体执行预编译合约
         return RunPrecompiledContract(p, input, contract)
      }
   }
```

下面是执行预编译合约的的具体内容

```go
func RunPrecompiledContract(p PrecompiledContract, input []byte, contract *Contract) (ret []byte, err error) {
   gas := p.RequiredGas(input)
   // 调用RequiredGas方法，UseGas扣除预编译合约消耗的gas
   if contract.UseGas(gas) {
      // Run方法具体执行预编译合约内容
      return p.Run(input)
   }
   return nil, ErrOutOfGas
}
```

下面是预编译合约的数据结构

```go
type PrecompiledContract interface {
   RequiredGas(input []byte) uint64  // RequiredPrice calculates the contract gas use
   Run(input []byte) ([]byte, error) // Run runs the precompiled contract
}
```

下面是在以太坊Homestead阶段内置的预编译合约

```go
var PrecompiledContractsHomestead = map[common.Address]PrecompiledContract{
   common.BytesToAddress([]byte{1}): &ecrecover{},
   common.BytesToAddress([]byte{2}): &sha256hash{},
   common.BytesToAddress([]byte{3}): &ripemd160hash{},
   common.BytesToAddress([]byte{4}): &dataCopy{},
}
```

下面看一下sha256hash合约的具体实现，根据数据结构中的定义这里定义了两个方法

```go
// SHA256 implemented as a native contract.
type sha256hash struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *sha256hash) RequiredGas(input []byte) uint64 {
   return uint64(len(input)+31)/32*params.Sha256PerWordGas + params.Sha256BaseGas
}
func (c *sha256hash) Run(input []byte) ([]byte, error) {
   h := sha256.Sum256(input)
   return h[:], nil
}
```



## 