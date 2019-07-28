<template>
  <v-container>
    <v-text-field
      label="Label"
      color="primary"
      v-model="query"
      @keypress.enter="getList"
    >
      <template v-slot:append>
        <v-btn depressed tile color="primary" class="ma-0" @click="getList">
          show
        </v-btn>
      </template>
    </v-text-field>
    <v-data-table :headers="headers" :items="list"> </v-data-table>
    {{ this.list }}
  </v-container>
</template>

<script lang="ts">
import { Component, Vue } from "vue-property-decorator";
import axios from "axios";
class WordCount {
  word: string;
  count: number;
  constructor(word: string, count: number) {
    this.word = word;
    this.count = count;
  }
}
class Header {
  text: string;
  value: string;
  constructor(text: string, value: string) {
    this.text = text;
    this.value = value;
  }
}
@Component
export default class HelloWorld extends Vue {
  query: string = "";
  headers: Header[] = [
    new Header("word", "word"),
    new Header("count", "count")
  ];
  list: WordCount[] = [];
  getList(): void {
    axios
      .post("/api/search", {
        query: this.query,
        maxResults: 100,
        fromDate: "200806280000",
        toDate: "201907260000"
      })
      .then(response => {
        this.list = response.data.top_words;
      });
  }
}
</script>

<style></style>
