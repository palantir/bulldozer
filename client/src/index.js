require('normalize.css');
require('@blueprintjs/core/dist/blueprint.css');
require('github-bot-ui/github-bot-ui.css');

var React = require('react');
var ReactDOM = require('react-dom');
var GitHubBotUi = require('github-bot-ui');

ReactDOM.render(
  React.createElement(GitHubBotUi, {
    appName: 'Bulldozer',
    docsUrl: "https://github.com/palantir/bulldozer#bulldozer"
  }),
  document.getElementById('container')
);
