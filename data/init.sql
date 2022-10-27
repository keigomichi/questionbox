CREATE DATABASE `questionbox` DEFAULT CHARACTER SET utf8mb4;

USE `questionbox`;

DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `questions`;
DROP TABLE IF EXISTS `posts`;

CREATE TABLE `users` (
  `Username` varchar(255) NOT NULL,
  `HashedPass` varchar(255) NOT NULL,
  PRIMARY KEY (`Username`)
);

CREATE TABLE `questions` (
  `ID` int NOT NULL AUTO_INCREMENT,
  `CONTENT` char(255) NOT NULL,
  PRIMARY KEY (`ID`)
);

CREATE TABLE `posts` (
  `ID` int NOT NULL AUTO_INCREMENT,
  `CONTENT` varchar(255) NOT NULL,
  `QuestionID` int NOT NULL,
  PRIMARY KEY (`ID`),
  FOREIGN KEY (`QuestionID`) REFERENCES `questions` (`ID`)
);