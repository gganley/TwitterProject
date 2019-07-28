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
    <v-text-field v-model="start" :rules="[date, required]" label="start"></v-text-field>
    <v-text-field v-model="stop" :rules="[date, required]" label="stop"></v-text-field>
    <v-data-table :headers="headers" :items="list"> </v-data-table>
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
start: string = "";
stop: string = "";
required(value: string): boolean | string {
  return !!value || "Required";
}
date(value: string): boolean | string {
const pattern = /20[0,1][0-9][0,1][0-9][0-3][0-9][0-2][0-9][0-5][0-9]/ ;
return pattern.test(value) || "YYYYMMDDHHmm";
}
  list: WordCount[] = [];
  getList(): void {
    axios
      .post("/api/search/file", {
        query: this.query,
        maxResults: 100,
fromDate: this.start,
toDate: this.stop
      })
      .then(response => {
        this.list = response.data.top_words;
      });
  }
}
</script>

<style></style>
