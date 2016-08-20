## Habits Tracker

Use Google Spreadsheets to program habits on different frequencies, i.e. daily, weekly, monthly or yearly. This code will fetch your data and create items on your Todoist account.

Additionally, it also fetches your results from Todoist and stores the results in a different sheet.

I am deploying this as an AWS Lambda service, it'll trigger every evening.

### Usage:

You need to set up a Google Spreadsheet. The main spreadsheet must be called *Habits*, the first row of this table will have the name of the columns and the rest of the rows will represent records.

#### Columns:

* *Habit*: This will be the name of the habit you want to register on Todoist. *Required*
* *Frequency*: Not the frequency itself but the unit of time from which you'll build your frequency. Choose one of "day", "week", "month", or "year". *Required*
* *Interval*: The number of units for each iteration. The actual frequency is calculated with Interval * Frequency.
* *Time*: The time of the day for a reminder. This has to be in the 00:00 form, i.e. half past 9 in the evening will be 21:30.
* *Next Iteration*: The next time you will have to act on the habit. Write down in the format of *1 January 2016*, once you set this, the program will automatically update it.

You will also need to create sheets titled "day", "week", "month" and "year". Leave the first column empty and the first row should contain the names of the habits. The program will add your results there.

If you want help to set this up for yourself, mail me at hi@faure.hu

### Caveats:

It's impossible to set a habit for the last date of the month. If set to August 31st, a month iteration will set it to October 1st. Have this in consideration in February.

### Motivation:
Todoist is a great To Do application and it supports iterative items but at the moment it is not possible to set reminders for every new iteration. I have been meaning to learn Go and play with AWS Lambdas so I decided this would be a fitting project.
