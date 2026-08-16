import { defineConfig } from 'wxt';
export default defineConfig({
  modules:['@wxt-dev/module-react'],
  manifest:{
    name:'MindBay Companion',description:'Amazon.de-Produkte kontrolliert in MindBay erfassen.',version:'0.1.0',
    permissions:['storage'],
    host_permissions:['https://www.amazon.de/*','https://amazon.de/*','http://127.0.0.1:8080/*','http://localhost:8080/*'],
    options_ui:{page:'options.html',open_in_tab:true},
  },
});
