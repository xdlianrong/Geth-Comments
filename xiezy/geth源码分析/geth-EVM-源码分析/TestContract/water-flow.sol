pragma solidity ^0.5.0;

contract watertap {
	// 向合约地址申请转账ETH
	uint256 a; 
	constructor(uint con)public{
	    a = con;
	}
    function getEth(uint amount) public {
        require(amount < 30000000000);
        msg.sender.transfer(amount);
    }
    // 给合约地址转账ETH
    function send() public payable {
    }
}