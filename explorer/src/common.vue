<script>
import Web3 from 'web3'

var web3 = new Web3()
var ethnodeurl = 'http://localhost:8545' // TODO: remote URL
web3.setProvider(new web3.providers.HttpProvider(ethnodeurl))

export default {
  web3
}
</script>
