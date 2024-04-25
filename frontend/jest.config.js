const config = {
  moduleNameMapper: {
    '\\.(scss|css)$': '<rootDir>/tests/style-mock.js'
  },
  setupFilesAfterEnv: ['<rootDir>/tests/setup-tests.ts'],
  testEnvironment: 'jsdom',
  transformIgnorePatterns: ['<rootDir>/node_modules/(?!pretty-bytes)']
};

module.exports = config;
