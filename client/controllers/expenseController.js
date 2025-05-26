
const Expense = require('../models/Expense');
const Group = require('../models/Group');

exports.addExpense = async (req, res) => {
  const { groupId, description, amount, payer, splitBetween, splitAmounts } = req.body;

  if (!groupId || !description || !amount || !payer || !splitBetween || !splitAmounts) {
    return res.status(400).json({ message: 'All fields including split info are required' });
  }

  if (!Array.isArray(splitBetween) || !Array.isArray(splitAmounts)) {
    return res.status(400).json({ message: 'Split info must be arrays' });
  }

  if (splitBetween.length !== splitAmounts.length) {
    return res.status(400).json({ message: 'Split arrays length mismatch' });
  }

  try {
    const group = await Group.findById(groupId);
    if (!group) return res.status(404).json({ message: 'Group not found' });

    // Check payer in group
    if (!group.members.some(m => m.toString() === payer)) {
      return res.status(400).json({ message: 'Payer not in group' });
    }

    // Check all splitBetween users are in group
    for (const userId of splitBetween) {
      if (!group.members.some(m => m.toString() === userId)) {
        return res.status(400).json({ message: `User ${userId} in splitBetween not in group` });
      }
    }

    // Validate sum of splitAmounts approximately equals amount (allow minor floating error)
    const sumSplits = splitAmounts.reduce((acc, val) => acc + val, 0);
    if (Math.abs(sumSplits - amount) > 0.01) {
      return res.status(400).json({ message: 'Sum of split amounts must equal total amount' });
    }

    const expense = new Expense({
      group: groupId,
      description,
      amount,
      paidBy: payer,
      splitBetween,
      splitAmounts,
    });

    await expense.save();

    return res.status(201).json({ message: 'Expense added', expense });
  } catch (err) {
    console.error(err);
    return res.status(500).json({ message: 'Server error' });
  }
};

exports.getExpensesByGroup = async (req, res) => {
  const { groupId } = req.params;
  try {
    const expenses = await Expense.find({ group: groupId }).lean();
    return res.json(expenses);
  } catch (err) {
    console.error(err);
    return res.status(500).json({ message: 'Server error' });
  }
};
 