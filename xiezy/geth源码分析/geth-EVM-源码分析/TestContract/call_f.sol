pragma solidity ^0.4.0; 
contract A {    
	address public temp1;   
	uint256 public temp2;    
	function three_call(address addr) public {        
		addr.call(bytes4(keccak256("test()")));      
		addr.delegatecall(bytes4(keccak256("test()")));    
		addr.callcode(bytes4(keccak256("test()")));     
	} 
} 

contract B {    
	address public temp1;    
	uint256 public temp2;    
	function test() public  {        
		temp1 = msg.sender;        
		temp2 = 100;    
	} 
}