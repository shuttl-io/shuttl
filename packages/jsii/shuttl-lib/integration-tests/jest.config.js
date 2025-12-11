/** @type {import('ts-jest').JestConfigWithTsJest} */
module.exports = {
  displayName: 'shuttl-lib-integration',
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>'],
  testMatch: ['**/*.integration.test.ts'],
  moduleFileExtensions: ['ts', 'js', 'json'],
  transform: {
    '^.+\\.ts$': ['ts-jest', {
      tsconfig: '<rootDir>/../tsconfig.json',
    }],
  },
  // Integration tests may take longer
  testTimeout: 60000,
  // Run tests sequentially since they spawn processes
  maxWorkers: 1,
};











