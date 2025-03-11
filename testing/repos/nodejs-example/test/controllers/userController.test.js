const userController = require('../../src/controllers/userController');

describe('UserController Tests', () => {
    describe('getUserById', () => {
        it('should return user information for valid ID', () => {
            const user = userController.getUserById(1);
            expect(user).toEqual({
                id: 1,
                name: 'Test User',
                email: 'test@example.com'
            });
        });

        it('should throw error for invalid ID', () => {
            expect(() => {
                userController.getUserById('invalid');
            }).toThrow('Invalid user ID');
        });
    });

    describe('validateEmail', () => {
        it('should return true for valid email', () => {
            expect(userController.validateEmail('test@example.com')).toBe(true);
        });

        it('should return false for invalid email', () => {
            expect(userController.validateEmail('invalid-email')).toBe(false);
        });
    });
}); 
