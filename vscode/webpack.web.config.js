// Webpack config for the laboratory web build. Same React tree as
// the extension's webview, but with vscode-api swapped out for the
// HTTP + WebSocket shim in src/web/web-api.ts.
//
// Output goes to ../cmd/am-server/web/, where Go's embed.FS picks it
// up for `am serve`. The HTML shell + bundle land together so a
// single binary serves the lab.
const path = require('path');
const CopyPlugin = require('copy-webpack-plugin');

module.exports = {
  target: 'web',
  entry: './src/web/index.tsx',
  output: {
    path: path.resolve(__dirname, '..', 'cmd', 'am-server', 'web'),
    filename: 'web.bundle.js',
    publicPath: './',
    clean: true,
  },
  resolve: {
    extensions: ['.tsx', '.ts', '.js'],
    // Swap the vscode-api implementation. Every React component
    // continues to `import { vscode } from '../vscode-api'`; webpack
    // routes that to the lab's HTTP+WS shim instead of the extension's
    // postMessage shim.
    alias: {
      [path.resolve(__dirname, 'src/webview/vscode-api')]: path.resolve(__dirname, 'src/web/web-api.ts'),
    },
  },
  module: {
    rules: [
      { test: /\.tsx?$/, exclude: /node_modules/, use: 'ts-loader' },
      { test: /\.css$/, use: ['style-loader', 'css-loader'] },
    ],
  },
  plugins: [
    new CopyPlugin({
      patterns: [
        { from: 'src/web/index.html', to: 'index.html', force: true },
      ],
    }),
  ],
  devtool: 'source-map',
};
