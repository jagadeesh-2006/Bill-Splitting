const Group = require('../models/Group');
const User = require('../models/User');

exports.createGroup = async (req, res) => {
  const { name, memberPhones, creatorId } = req.body;
  if (!name || !memberPhones || !Array.isArray(memberPhones) || memberPhones.length === 0 || !creatorId) {
    return res.status(400).json({ message: 'Group name, member phones and creatorId are required' });
  }

  try {
    // find users by phone numbers
    const users = await User.find({ phone: { $in: memberPhones } });
    if (users.length === 0) return res.status(400).json({ message: 'No users found with those phone numbers' });

    const memberIds = users.map(u => u._id.toString());

    // Make sure creatorId is included as member
    if (!memberIds.includes(creatorId)) {
      memberIds.push(creatorId);
    }

    const group = new Group({
      name,
      members: memberIds,
    });

    await group.save();
    res.status(201).json({ message: 'Group created', group });
  } catch (err) {
    console.error(err);
    res.status(500).json({ message: 'Server error' });
  }
};

exports.getUserGroups = async (req, res) => {
  const userId = req.params.userId;
  try {
    const groups = await Group.find({ members: userId }).populate('members', 'username email phone');
    res.json(groups);
  } catch (err) {
    res.status(500).json({ message: 'Server error' });
  }
};
