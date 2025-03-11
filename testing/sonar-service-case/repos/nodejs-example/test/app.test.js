const request = require('supertest');
const app = require('../src/app');

// Test suite for the main app
describe('App Tests', () => {
    // Test the root endpoint
    describe('GET /', () => {
        it('should return welcome message', async () => {
            const response = await request(app)
                .get('/')
                .expect('Content-Type', /json/)
                .expect(200);

            expect(response.body).toEqual({
                message: 'Welcome to Sonar Test Project'
            });
        });
    });
}); 
