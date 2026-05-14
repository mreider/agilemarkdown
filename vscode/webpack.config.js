// Webpack config for the Agile Markdown VS Code extension. Two bundles:
//   1. extension.js — Node target, runs in the extension host.
//   2. webview.js   — web target, served from the webview iframe.
const path = require('path');

const extensionConfig = {
  target: 'node',
  entry: './src/extension.ts',
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: 'extension.js',
    libraryTarget: 'commonjs2',
    clean: false,
  },
  externals: { vscode: 'commonjs vscode' },
  resolve: { extensions: ['.ts', '.tsx', '.js'] },
  module: {
    rules: [
      { test: /\.tsx?$/, exclude: /node_modules/, use: 'ts-loader' },
    ],
  },
  devtool: 'source-map',
};

const webviewConfig = {
  target: 'web',
  entry: './src/webview/index.tsx',
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: 'webview.js',
  },
  resolve: { extensions: ['.tsx', '.ts', '.js'] },
  module: {
    rules: [
      { test: /\.tsx?$/, exclude: /node_modules/, use: 'ts-loader' },
      { test: /\.css$/, use: ['style-loader', 'css-loader'] },
    ],
  },
  devtool: 'source-map',
};

module.exports = [extensionConfig, webviewConfig];
