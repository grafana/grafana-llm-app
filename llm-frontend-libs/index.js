'use strict';

if (process.env.NODE_ENV === 'production') {
  module.exports = require('./dist/index.production.js');
} else {
  module.exports = require('./dist/index.development.js');
}
