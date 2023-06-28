# Logstash analysis

Code in this repo (in the logs directory) is used to look at the errors in logstash logs
and come up with counts of the different errors observed, and also counts of such errors per day.

The reason for doing this is that as of 7th June 2023 there are
far to many errors in the logs and we need a way of determining
if any "fixes" are making a difference.

The output of the scripts enable us to have a clear view of the distribution of each kind of log line.

By running the code and recording the results, making some changes/fixes ... then waiting 24 hours and running the analysis again ...
We can look at the daily counts of the last 24 hours and compare to previous days to see if there is any
improvement (reduction) in the numbers of errors seen.

-=-=-

First select the environment from where you want to get the logs from.

1. cd into 'logs' and run the script, which will take a while and you will be prompted to answer 'yes':

  get-all-logs.sh

2. Then run script:
  
  process-logs.py

  NOTE: This script may fail - indicating a new log that it does not recognise.
        If that happens then update the function in it called 'process_line' accordingly.
        Then run the script again, and if it identifies another unknown error,
        repeat this process until it runs to completion.

3. Scrutinize the output

   It starts with a summary of the counts for all error's, for example:

running_error_counts: 0 is: 461654
running_error_counts: 1 is: 465625
running_error_counts: 2 is: 271964
running_error_counts: 3 is: 465618
running_error_counts: 4 is: 937
running_error_counts: 5 is: 812
running_error_counts: 6 is: 937
running_error_counts: 7 is: 234472
running_error_counts: 8 is: 5
running_error_counts: 9 is: 7
running_error_counts: 10 is: 7
running_error_counts: 11 is: 7
.
.
. and so on.


   Then ...

   Here's a small sample of what the output can look like (without colour highlighting)

   (the lines are over 200 chars wide, to expand the width of the view until line wrap goes away)

   The 'Error numbers' row are column headings of the error number to match the numbers returned in the function
   'process_line'.
   The error counts in the terminal when the app is run are highlighted as:
      Yellow indicating more than the previous day
      Green indicating less than the previous day
      White indicating no change in count
   
   If there are no errors of a particular error type on a day, then no number '0' is displayed to aid clarity.

   (the below has no colour highlighting as its just plain text)

          Error numbers: [   0     1    2     3   4   5   6     7  8  9 10 11 12  13 14 15     16 17 18  19 20 21   22   23  24  25 26 27 28 29 30 31 32   33 34 35   36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56  57 58 ]
Date: 2023-05-29 Counts: [           2693         8   3   8   122                  3                      8                                                 8                                                                               ]
Date: 2023-05-30 Counts: [           2692         4   3   4   291                  3                      4                                                 4                                                                               ]
Date: 2023-05-31 Counts: [           2691         1       1   178                                         1                                                 1                                                                               ]
Date: 2023-06-01 Counts: [           2691         5   3   5   163                  3                      5                                                 5                                                                               ]
Date: 2023-06-02 Counts: [         2 2692     2   3   1   3   129                  1                      3                                                 3                                                                               ]
Date: 2023-06-03 Counts: [           2692         5   2   5   144                  2                      5                                                 5                                                                               ]
Date: 2023-06-04 Counts: [           2692         2       2   137                                         2                                                 2                                                                               ]
          Error numbers: [   0     1    2     3   4   5   6     7  8  9 10 11 12  13 14 15     16 17 18  19 20 21   22   23  24  25 26 27 28 29 30 31 32   33 34 35   36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56  57 58 ]
Date: 2023-06-05 Counts: [           2691         2       2   117                                         2                                                 2                                                                               ]
Date: 2023-06-06 Counts: [           2692         2   1   2   163                  1                      2                                                 2                                                                               ]
Date: 2023-06-07 Counts: [            935         2   1   2    93                  1                      2                                                 2                                                                               ]

4. When yo want to look into all of the log files, you can run: uncompress-all-to-three.sh

  This will uncompress all of the log files per box into one file per logstash box into these files:

    logs-1.txt
    logs-2.txt
    logs-3.txt

5. You should clean up the downloaded and created files once you are done.
   That is delete files in directories:
   logstash-1/logstash
   logstash-2/logstash
   logstash-3/logstash
  