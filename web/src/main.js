import { mount } from 'svelte'
import App from './App.svelte'

// Clear the static loading spinner before mounting Svelte
const target = document.getElementById('app')
target.innerHTML = ''

const app = mount(App, { target })

export default app
