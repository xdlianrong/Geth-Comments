<template>
  <div class="home" style="padding-top: 50px;">
    <h4 style="text-align:center; margin-top: 30px;">
      Latest Block: <router-link v-bind:to="'/block/' + blockNumber">{{ blockNumber }}</router-link>
    </h4>
    <table class="table" style="margin: auto" width='60%'>
      <tbody>
      <tr>
        <th>Block #</th>
        <th>Tx #</th>
        <th>Size</th>
        <th>Timestamp</th>
      </tr>
      <tr v-for="(i,index) in blocks" :key="index">
        <td>
          <router-link v-bind:to="'/block/' + i.number" exact>{{ i.number }}</router-link></td>
        <td>{{ i.transactions.length }}</td>
        <td>{{ i.size }}</td>
        <td>{{ i.timestamp }}</td>
      </tr>
      </tbody>
    </table>
    <router-view/>
<!--    <HelloWorld msg="Welcome to Your Vue.js App"/>-->
  </div>
</template>

<script>
// @ is an alias to /src
// 局部注册组件
// import HelloWorld from '@/components/HelloWorld.vue'
import common from '@/common'

export default {
  name: 'Home',
  // components: {
  //   HelloWorld
  // },
  data () {
    return {
      blocks: [],
      maxBlocks: 0,
      blockNumber: 0
    }
  },
  async created () {
    const web3 = common.web3
    this.maxBlocks = 50
    await web3.eth.getBlockNumber()
      .then((result) => {
        // console.log(result)
        this.blockNumber = result
        if (this.maxBlocks > result) {
          this.maxBlocks = result + 1
        }
        async function run (maxBlocks, blockNumber, blocks) {
          for (let i = 0; i < maxBlocks; i++) {
            // console.log(this.blockNumber)
            await web3.eth.getBlock(blockNumber - i)
              .then((result) => {
                // console.log(result)
                blocks.push(result)
              })
          }
        }
        run(this.maxBlocks, this.blockNumber, this.blocks)
      })
  }
}
</script>

<style type="text/css">
table {
  border-collapse: collapse;
  width: 50em;
  border: 1px solid #666;
}
th, td {
  padding: 0.1em 1em;
}
</style>
