import './style.css';
import { createApp } from 'vue';
import App from './app/App.vue';
import { start } from './app/boot';

createApp(App).mount('#app');
start();
