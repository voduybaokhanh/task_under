const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

// Add resolver alias for axios to use browser-compatible version
config.resolver.alias = {
  ...config.resolver.alias,
  'axios': require.resolve('axios/dist/browser/axios.cjs'),
};

module.exports = config;