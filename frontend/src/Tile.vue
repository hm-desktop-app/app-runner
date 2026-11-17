<script setup lang="ts">
import {computed, ref} from 'vue';
import type {App} from './state';
import {failedIcons, tileTone} from './ui';

const props = defineProps<{app: App; lazy?: boolean}>();
const loaded = ref(false);
const failed = ref(false);
const url = computed(() => props.app.iconURL || '');
const showImage = computed(() => !!url.value && !failed.value && !failedIcons.has(url.value));
const mark = computed(() => Array.from(props.app.title || '?')[0] || '?');
const tone = computed(() => tileTone(props.app.title));

function onError(): void {
 failedIcons.add(url.value);
 failed.value = true;
 loaded.value = false;
}
</script>

<template>
 <span class="tile" :class="{loaded}" :style="tone" aria-hidden="true">
  <span class="tile-mark">{{ mark }}</span>
  <img v-if="showImage" class="tile-icon" :src="url" alt="" decoding="async" :loading="lazy ? 'lazy' : undefined" @load="loaded = true" @error="onError">
 </span>
</template>
