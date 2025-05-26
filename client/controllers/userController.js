const User = require('../models/User');

exports.getAllUsers = async (req, res) => {
  try {
    const users = await User.find({}, 'username email phone');
    res.json(users);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
};
