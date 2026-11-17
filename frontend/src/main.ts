import './style.css';
import { createApp } from 'vue';
import App from './app/App.vue';
import { start } from './app/store';

createApp(App).mount('#app');
start();
