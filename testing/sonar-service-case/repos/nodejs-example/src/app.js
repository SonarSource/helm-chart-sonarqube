const express = require('express');
const userController = require('./controllers/userController');

// Initialize express app
const app = express();
const port = process.env.PORT || 3000;

// Basic route handler
app.get('/', (req, res) => {
    res.json({ message: 'Welcome to Sonar Test Project' });
});

// Add user route
app.get('/user/:id', (req, res) => {
    try {
        const userId = parseInt(req.params.id);
        const user = userController.getUserById(userId);
        res.json(user);
    } catch (error) {
        res.status(400).json({ error: error.message });
    }
});

// Start server
app.listen(port, () => {
    console.log(`Server is running on port ${port}`);
});

module.exports = app; 
