const path = require('path');
const webpack = require('webpack');

const ExtractTextPlugin = require('extract-text-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const CopyWebpackPlugin = require('copy-webpack-plugin');

const plugins = [
  new ExtractTextPlugin('[name]-[hash].css'),
  new HtmlWebpackPlugin({
    minify: {
      collapseWhitespace: true
    },
    template: path.resolve(__dirname, 'src/index.html'),
    title: 'Bulldozer'
  }),
  new CopyWebpackPlugin([{
      from: path.resolve(__dirname, "src/favicons"),
      to: path.resolve(__dirname, "build/favicons")
  }])
];

if (process.env.NODE_ENV === 'production') {
  plugins.push(
    new webpack.DefinePlugin({
      'process.env': {
        'NODE_ENV': '"production"'
      }
    }),
    new webpack.optimize.DedupePlugin(),
    new webpack.optimize.UglifyJsPlugin({
      compress: {
        warnings: false
      }
    }),
    new webpack.optimize.OccurenceOrderPlugin()
  );
}

module.exports = {
  entry: {
    'bulldozer': [
      path.resolve(__dirname, 'src/index.js')
    ]
  },
  output: {
    path: path.resolve(__dirname, 'build'),
    filename: '[name]-[hash].js',
    publicPath: '/'
  },
  module: {
    loaders: [{
      test: /\.css$/,
      loader: ExtractTextPlugin.extract('style', 'css')
    }, {
      test: /\.(woff|ttf|eot|svg|gif|jpeg|jpg|png)([\?]?.*)$/,
      loader: 'file',
      include: path.resolve(__dirname, 'node_modules'),
      query: {
        name: 'resources/[name].[ext]'
      }
    }]
  },
  plugins: plugins
};