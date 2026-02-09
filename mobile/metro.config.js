const { getDefaultConfig } = require("expo/metro-config");

const config = getDefaultConfig(__dirname);

config.resolver.resolverMainFields = ["react-native", "browser", "main"];

config.resolver.alias = {
  axios: require.resolve("axios/dist/browser/axios.cjs"),
};

module.exports = config;
