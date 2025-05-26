const express = require('express');
const router = express.Router();
const { createGroup, getUserGroups } = require('../controllers/groupController');

router.post('/create', createGroup);
router.get('/user-groups/:userId', getUserGroups);

module.exports = router;
