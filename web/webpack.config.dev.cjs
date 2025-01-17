const path = require("path");
const { CleanWebpackPlugin } = require("clean-webpack-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const FontConfigWebpackPlugin = require("font-config-webpack-plugin");
const { WebpackManifestPlugin } = require("webpack-manifest-plugin");

module.exports = {
  mode: "development",
  entry: "./static/js/main.js",
  output: {
    filename: "[name].[contenthash].js",
    path: path.resolve(__dirname, "static/dist"),
    clean: true,
  },
  module: {
    rules: [
      {
        test: /\.css$/,
        // use: ["style-loader", "css-loader"],
        // use: [MiniCssExtractPlugin.loader, "style-loader", "css-loader", "postcss-loader"],
        use: [MiniCssExtractPlugin.loader, "css-loader"],
      },
    ],
  },
  resolve: {
    alias: {
      "jquery-ui": "jquery-ui-dist/jquery-ui.js",
    },
  },
  plugins: [
    new CleanWebpackPlugin(),
    new HtmlWebpackPlugin({
      template: "./views/layouts/main.template.html",
      // scriptLoading: "blocking",
      // filename: "../../views/layouts/main.html",
      filename: "../main.html",
      // minify: false,
    }),
    new MiniCssExtractPlugin({
      filename: "[name].[contenthash].css",
    }),
    new WebpackManifestPlugin({
      fileName: "manifest.json",
      publicPath: "/static/dist/",
    }),
    new FontConfigWebpackPlugin(),
  ],
  devtool: "eval-source-map", // Fast rebuilds for development
  watchOptions: {
    aggregateTimeout: 600,
    ignored: ["**/dist", "**/node_modules", "/node_modules/"],
  },
  optimization: {
    moduleIds: "deterministic",
    runtimeChunk: "single",
    splitChunks: {
      // chunks: "all",
      cacheGroups: {
        vendor: {
          test: /[\\/]node_modules[\\/]/,
          name: "vendors",
          chunks: "all",
        },
      },
    },
  },
};
