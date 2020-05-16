pragma solidity ^0.5.0; 

contract B {    
	address public temp1;    
	uint256 public temp2;    
	function test() public  {        
		temp1 = msg.sender;        
		temp2 = 100;    
    }
}

contract C_call {    
	address public temp1;   
	uint256 public temp2;    
	function call_call(address addr) public {        
		addr.call(abi.encode(bytes4(keccak256("test()"))));        
    }
} 

contract C_delegatecall {    
	address public temp1;   
	uint256 public temp2;    
	function delegate_call(address addr) public {            
		addr.delegatecall(abi.encode(bytes4(keccak256("test()"))));       
    }
} 

contract C_high{
    address public temp1;   
	uint256 public temp2; 
	function high_call(address addr) public {
	    B(addr).test();
	}
}