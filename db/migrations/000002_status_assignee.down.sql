ALTER TABLE tasks DROP COLUMN assignee;
ALTER TABLE tasks RENAME COLUMN status TO done;
