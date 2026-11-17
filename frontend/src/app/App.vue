<script setup lang="ts">
import Sheet from '../sheet/Sheet.vue';
import CommandView from './CommandView.vue';
import LibraryView from './LibraryView.vue';
import Overlays from './Overlays.vue';
</script>

<template>
    <div class="window-body">
        <CommandView />
        <LibraryView />
    </div>
    <div id="sheet-root"><Sheet /></div>
    <Overlays />
</template>
