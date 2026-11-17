<script setup lang="ts">
import CommandView from './CommandView.vue';
import LibraryView from './LibraryView.vue';
import Overlays from './Overlays.vue';
</script>

<template>
 <div class="window-body">
  <CommandView />
  <LibraryView />
 </div>
 <Overlays />
</template>
