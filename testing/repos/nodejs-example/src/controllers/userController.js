/**
 * User controller for handling user-related operations
 */
class UserController {
    /**
     * Get user information by ID
     * @param {number} id - User ID
     * @returns {Object} User information
     */
    getUserById(id) {
        // 模拟数据库查询
        if (!id || typeof id !== 'number') {
            throw new Error('Invalid user ID');
        }
        
        return {
            id: id,
            name: 'Test User',
            email: 'test@example.com'
        };
    }

    /**
     * Validate user email format
     * @param {string} email - Email to validate
     * @returns {boolean} True if email is valid
     */
    validateEmail(email) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    }
}

module.exports = new UserController(); 
