const express = require('express');
const cors = require('cors');
const connectDB = require('./config/db'); // Adjust path if different

const app = express();
app.use(cors());
app.use(express.json());


// Routes
const authRoutes = require('./routes/authRoute');
const userRoutes = require('./routes/userRoute');
const groupRoutes = require('./routes/groupRoute');
const expenseRoutes = require('./routes/expenseRoute');

app.use('/api/auth', authRoutes);
app.use('/api/users', userRoutes);
app.use('/api/groups', groupRoutes);
app.use('/api/expenses', expenseRoutes);


// Start server after DB is connected
connectDB().then(() => {
  app.listen(5000, () => {
    console.log('Server running on port 5000');
  });
});
