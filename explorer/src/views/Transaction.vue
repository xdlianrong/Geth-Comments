<template>
  <div class="transaction">
    <table style="margin: auto" width='60%'>
      <tbody><tr>
        <th colspan="2" style="text-align:left">Summary</th>
      </tr>
      <tr>
        <td>Block Hash</td>
        <td>{{transaction.blockHash}}</td>
      </tr>
      <tr>
        <td>Included In Block</td>
        <td>
          <router-link v-bind:to="'/block/' + transaction.blockNumber">{{ transaction.blockNumber }}</router-link>
        </td>
      </tr>
      <tr>
        <td>Gas Used</td>
        <td>{{transaction.gas}}</td>
      </tr>
      <tr>
        <td>Gas Price</td>
        <td>{{transaction.gasPrice}}</td>
      </tr>
      <tr>
        <td>Number of transactions made by the sender prior to this one</td>
        <td>{{transaction.nonce}}</td>
      </tr>
      <tr>
        <td>Transaction price</td>
        <td>{{(transaction.gas * transaction.gasPrice)/1000000000000000000 + " ETH"}}</td>
      </tr>
      <tr>
        <td>Data</td>
        <td>{{transaction.input}}</td>
      </tr>
      <tr>
        <td>CmO</td>
        <td>{{transaction.CmO}}</td>
      </tr>
      <tr>
        <td>CmR</td>
        <td>{{transaction.CmR}}</td>
      </tr>
      <tr>
        <td>CmRpk</td>
        <td>{{transaction.CmRpk}}</td>
      </tr>
      <tr>
        <td>CmV</td>
        <td>{{transaction.CmV}}</td>
      </tr>
      <tr>
        <td>Sig</td>
        <td>{{transaction.Sig}}</td>
      </tr>

      </tbody>
    </table>
  </div>
</template>

<script>

import common from '@/common'

export default {
  name: 'Transaction',
  data () {
    return {
      transaction: {},
      transactionhash: this.$route.params.transactionhash
    }
  },
  async created () {
    const web3 = common.web3
    await web3.eth.getTransaction(this.transactionhash)
      .then((result) => {
        this.transaction = result
      })
  }

}
</script>
<style>
td{
  border-bottom:1px solid black;
}
</style>
