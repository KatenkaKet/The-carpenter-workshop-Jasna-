import {defineConfig} from '@playwright/test';
export default defineConfig({testDir:'./tests',timeout:90000,workers:1,use:{headless:true,actionTimeout:15000,navigationTimeout:15000,trace:'retain-on-failure'},reporter:'list'});
