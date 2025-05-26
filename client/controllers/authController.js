const User = require('../models/User');
const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');
// const dotenv = require('dotenv');
// dotenv.config();

exports.register = async (req, res) => {
  const { username, email, password, phone } = req.body;
  try {
    let user = await User.findOne({ email });
    if (user) return res.status(400).json({ message: 'Email already registered' });

    user = await User.findOne({ phone });
    if (user) return res.status(400).json({ message: 'Phone number already registered' });

    const hashedPassword = await bcrypt.hash(password, 10);

    user = new User({ username, email, password: hashedPassword, phone });
    await user.save();

    res.status(201).json({ message: 'User registered successfully' });
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
};

exports.login = async (req, res) => {
  const { email, password } = req.body;
  try {
    const user = await User.findOne({ email });
    if (!user) return res.status(400).json({ message: 'Invalid credentials' });

    const isMatch = await bcrypt.compare(password, user.password);
    if (!isMatch) return res.status(400).json({ message: 'Invalid credentials' });

    const payload = { user: { id: user._id } };
    const token = jwt.sign(payload, 'ce5ccc3afe5ff839b845b3ad8568b6b6b711fd663c5fe518857dc811b8ccb2ac96af2416b', { expiresIn: '12h' });


    res.json({
      token,
      user: {
        _id: user._id,
        username: user.username,
        email: user.email,
        phone: user.phone,
      }
    });
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
};
