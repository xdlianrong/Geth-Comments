<template>
  <div class="block">
<!--    <p>{{ block }}</p>-->
    <table style="margin: auto" width='60%'>
      <tbody>
      <tr>
        <th colspan="2" style="text-align:left">Summary</th>
      </tr>
      <tr>
        <td>Block Number</td>
        <td>{{ block.number }}</td>
      </tr>
      <tr>
        <td>Received Time</td>
        <td>
          {{ block.timestamp }}
        </td>
      </tr>
      <tr>
        <td>Difficulty</td>
        <td>
          {{ block.difficulty }}
        </td>
      </tr>
      <tr>
        <td>Nonce</td>
        <td>{{ block.nonce }}</td>
      </tr>
      <tr>
        <td>Size</td>
        <td>{{ block.size }}</td>
      </tr>
      <tr>
        <td>Miner</td>
        <td>{{ block.miner }}</td>
      </tr>
      <tr>
        <td>Gas Limit</td>
        <td>{{ block.gasLimit }}</td>
      </tr>
      <tr>
        <td>Data</td>
        <td>{{ block.extraData }}</td>
      </tr>

      </tbody>
    </table>
    <div v-for="(tx,index) in transactions" :key="index">
      <hr>
      <table style="margin: auto" width='60%'>
        <tbody>
        <tr>
          <th colspan="2" style="text-align:left">Transaction #{{index+1}}</th>
        </tr>
        <tr>
          <td>Hash #</td>
          <td><router-link v-bind:to="'/transaction/' + tx.hash">{{ tx.hash }}</router-link></td>
        </tr>
        <tr>
          <td>From</td>
          <td>{{tx.from}}</td>
        </tr>
        <tr>
          <td>To</td>
          <td>{{tx.to}}</td>
        </tr>
        <tr>
          <td>Gas</td>
          <td>{{tx.gas}}</td>
        </tr>
        <tr>
          <td>Input</td>
          <td>{{tx.input}}</td>
        </tr>
        <tr>
          <td>Value</td>
          <td>{{tx.value}}</td>
        </tr>
        </tbody>
      </table>
      <hr>
    </div>
  </div>
</template>

<script>

import common from '@/common'

export default {
  name: 'block',
  data () {
    return {
      blocknum: this.$route.params.blocknum,
      block: {},
      transactions: []
    }
  },
  async created () {
    const web3 = common.web3
    await web3.eth.getBlock(this.blocknum)
      .then((result) => {
        console.log(result)
        this.block = result
      })

    await web3.eth.getBlockTransactionCount(this.block.number)
      .then((result) => {
        const txCount = result
        async function run (txCount, blocknumber, transactions) {
          for (let blockIdx = 0; blockIdx < txCount; blockIdx++) {
            await web3.eth.getTransactionFromBlock(blocknumber, blockIdx)
              .then((result) => {
                var transaction = {
                  id: result.hash,
                  hash: result.hash,
                  from: result.from,
                  to: result.to,
                  gas: result.gas,
                  input: result.input,
                  value: result.value
                }
                transactions.push(transaction)
              })
          }
        }
        run(txCount, this.block.number, this.transactions)
      })
  }

}
</script>
