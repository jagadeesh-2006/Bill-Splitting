const mongoose = require('mongoose');
const Schema = mongoose.Schema;

const expenseSchema = new Schema({
  group: { type: Schema.Types.ObjectId, ref: 'Group', required: true },
  description: { type: String, required: true },
  amount: { type: Number, required: true },
  paidBy: { type: Schema.Types.ObjectId, ref: 'User', required: true },
  splitBetween: [{ type: Schema.Types.ObjectId, ref: 'User', required: true }],
  splitAmounts: [{ type: Number, required: true }],
}, { timestamps: true });

module.exports = mongoose.model('Expense', expenseSchema);
