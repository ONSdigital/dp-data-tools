#!/usr/bin/python3

import sys
import os
from pathlib import Path
import gzip
from dateutil.parser import parse


MAX_ERRORS = 100

def process_line(line):
    # Return an error number to indicate which error type has been seen, or exit if event is unknown for code to be adjusted

    # The strings below are the parts of the log line that do not have varying bits of info in them like date or url

    # First process all lines with the words 'error' or 'ERROR' in it:
    if "[WARN ][logstash.outputs.amazonelasticsearch] Attempted to resurrect connection to dead ES instance, but got an error." in line:
        return 0
    if "[ERROR][logstash.javapipeline    ] A plugin had an unrecoverable error. Will restart this plugin." in line:
        return 1
    if "[ERROR][logstash.outputs.amazonelasticsearch] Encountered a retryable error. Will Retry with exponential backoff" in line:
        return 2
    if "Stack: /usr/share/logstash/vendor/bundle/jruby/2.5.0/gems/aws-sdk-core-2.11.632/lib/seahorse/client/plugins/raise_response_errors.rb:15:in `call'" in line:
        return 3
    if "[ERROR][logstash.outputs.amazonelasticsearch] Attempted to send a bulk request to elasticsearch' but Elasticsearch appears to be unreachable or down!" in line:
        return 4
    if "[ERROR][logstash.outputs.amazonelasticsearch] Attempted to send a bulk request to elasticsearch, but no there are no living connections in the connection pool. Perhaps Elasticsearch is unreachable or down?" in line:
        return 5
    if "[WARN ][logstash.outputs.amazonelasticsearch] Marking url as dead. Last error: [LogStash::Outputs::AmazonElasticSearch::HttpClient::Pool::HostUnreachableError] Elasticsearch Unreachable:" in line:
        return 6
    if "[ERROR][logstash.filters.ruby    ] Ruby exception occurred: bad URI(is not URI?):" in line:
        # examples of the above:
        # [ERROR][logstash.filters.ruby    ] Ruby exception occurred: bad URI(is not URI?): http://web-prod-149552176.eu-west-2.elb.amazonaws.com:80-
        # [ERROR][logstash.filters.ruby    ] Ruby exception occurred: bad URI(is not URI?): https://www.ons.gov.uk:443/employmentandlabourmarket/peopleinwork/earningsandworkinghours/datasets/regionbyindustry2digitsicashetable5\x22
        # [ERROR][logstash.filters.ruby    ] Ruby exception occurred: bad URI(is not URI?): http://web-prod-149552176.eu-west-2.elb.amazonaws.com:443-
        return 7
    if '[ERROR][logstash.outputs.amazonelasticsearch] An unknown error occurred sending a bulk request to Elasticsearch. We will retry indefinitely {:error_message=>"no implicit conversion of nil into String"' in line:
        return 8
    if "/usr/share/logstash/vendor/bundle/jruby/2.5.0/gems/aws-sdk-core-2.11.632/lib/aws-sdk-core/xml/error_handler.rb:8:in `call'" in line:
        return 9
    if "/usr/share/logstash/vendor/bundle/jruby/2.5.0/gems/aws-sdk-core-2.11.632/lib/aws-sdk-core/plugins/helpful_socket_errors.rb:10:in `call'" in line:
        return 10
    if "/usr/share/logstash/vendor/bundle/jruby/2.5.0/gems/aws-sdk-core-2.11.632/lib/aws-sdk-core/plugins/retry_errors.rb:108:in `call'" in line:
        return 11
    if "/usr/share/logstash/vendor/bundle/jruby/2.5.0/gems/aws-sdk-core-2.11.632/lib/seahorse/client/plugins/raise_response_errors.rb:14:in `call'" in line:
        return 12
    if "[WARN ][logstash.outputs.amazonelasticsearch] UNEXPECTED POOL ERROR" in line:
        return 13
    if '[ERROR][org.logstash.execution.ShutdownWatcherExt] The shutdown process appears to be stalled due to busy or blocked plugins. Check the logs for more information.' in line:
        return 14
    if "[ERROR][logstash.filters.ruby    ] Ruby exception occurred: undefined method `split' for nil:NilClass" in line:
        return 15
    
    # now process 'WARN':
    if '[WARN ][logstash.inputs.s3       ] Unable to download remote file {:exception=>Aws::S3::Errors::NoSuchKey, :message=>"The specified key does not exist."' in line:
        return 16
    if '[WARN ][logstash.filters.json    ] Exception caught in json filter {:exception=>"Invalid FieldReference:' in line:
        return 17
    if '[WARN ][logstash.filters.json    ] Parsed JSON object/hash requires a target configuration option' in line:
        return 18
    if '[WARN ][logstash.outputs.amazonelasticsearch] Restored connection to ES instance' in line:
        return 19
    if '[WARN ][logstash.runner          ] SIGHUP received' in line:
        return 20
    if '[WARN ][org.logstash.input.DeadLetterQueueInputPlugin] closing dead letter queue input plugin' in line:
        return 21
    if '[WARN ][io.netty.channel.AbstractChannelHandlerContext] Failed to submit an exceptionCaught() event' in line:
        return 22
    if '[WARN ][io.netty.channel.AbstractChannelHandlerContext] The exceptionCaught() event that was failed to submit was:' in line:
        return 23
    if "[WARN ][io.netty.channel.AbstractChannelHandlerContext] An exception 'java.lang.NullPointerException' [enable DEBUG level for full stacktrace] was thrown by a user handler's exceptionCaught() method while handling the following exception:" in line:
        return 24
    if '[WARN ][io.netty.util.concurrent.AbstractEventExecutor] A task raised an exception. Task:' in line:
        return 25
    if '[WARN ][logstash.inputs.s3       ] Logstash S3 input, stop reading in the middle of the file, we will read it again when logstash is started' in line:
        return 26
    if '[WARN ][org.logstash.execution.ShutdownWatcherExt] {"' in line:
        return 27
    if "[WARN ][logstash.outputs.amazonelasticsearch] Detected a 6.x and above cluster: the `type` event field won't be used to determine the document _type {:es_version=>7}" in line:
        return 28
    if "[WARN ][logstash.config.source.multilocal] Ignoring the 'pipelines.yml' file because modules or command line options are specified" in line:
        return 29
    if '[WARN ][io.netty.channel.DefaultChannelPipeline] An exceptionCaught() event was fired, and it reached at the tail of the pipeline. It usually means the last handler in the pipeline did not handle the exception.' in line:
        return 30
    if '[WARN ][logstash.runner          ] SIGINT received. Shutting down.' in line:
        return 31
    if '[WARN ][logstash.runner          ] Received shutdown signal, but pipeline is still waiting for in-flight events' in line:
        return 32

    # now process 'INFO':
    if '[INFO ][logstash.outputs.amazonelasticsearch] Running health check to see if an Elasticsearch connection is working' in line:
        return 33
    if '[INFO ][logstash.pipelineaction.reload] Reloading pipeline {"pipeline.id"=>:main}' in line:
        return 34
    if '[INFO ][org.logstash.input.DeadLetterQueueInputPlugin] writing offsets' in line:
        return 35
    if '[INFO ][org.logstash.beats.BeatsHandler] [local: 0.0.0.0:5044, remote:' in line:
        return 36
    if '[INFO ][logstash.javapipeline    ] Pipeline terminated {"pipeline.id"=>"main"}' in line:
        return 37
    if '[INFO ][logstash.outputs.amazonelasticsearch] Elasticsearch pool URLs updated' in line:
        return 38
    if '[INFO ][logstash.outputs.amazonelasticsearch] ES Output version determined' in line:
        return 39
    if '[INFO ][logstash.outputs.amazonelasticsearch] New Elasticsearch output' in line:
        return 40
    if '[INFO ][logstash.javapipeline    ] Starting pipeline {' in line:
        return 41
    if '[INFO ][logstash.javapipeline    ] Pipeline Java execution initialization time' in line:
        return 42
    if '[INFO ][logstash.inputs.s3       ] Registering {:bucket=>"ons-dp-prod-elb-logs", :region=>"eu-west-2"}' in line:
        return 43
    if '[INFO ][logstash.inputs.s3       ] Registering {:bucket=>"ons-dp-prod-flow-logs", :region=>"eu-west-2"}' in line:
        return 44
    if '[INFO ][logstash.inputs.beats    ] Starting input listener {:address=>"0.0.0.0:5044"}' in line:
        return 45
    if '[INFO ][logstash.javapipeline    ] Pipeline started {"pipeline.id"=>"main"}' in line:
        return 46
    if '[INFO ][org.logstash.beats.Server] Starting server on port: 5044' in line:
        return 47
    if '[INFO ][logstash.agent           ] Pipelines running {:count=>1, :running_pipelines=>[:main], :non_running_pipelines=>[]}' in line:
        return 48
    if '[INFO ][logstash.inputs.s3       ] Using default generated file for the sincedb {:filename=>' in line:
        return 49
    if '[INFO ][logstash.runner          ] Logstash shut down.' in line:
        return 50
    if '[logstash.runner          ] Starting Logstash {"logstash.version"=>"7.11.2", "jruby.version"=>"jruby 9.2.13.0 (2.5.7) 2020-08-03 9a89c94bcc OpenJDK 64-Bit Server VM 11.0.8+10 on' in line:
        return 51
    if '[INFO ][org.reflections.Reflections] Reflections took' in line:
        return 52
    if '[INFO ][logstash.agent           ] Successfully started Logstash API endpoint {:port=>9600}' in line:
        return 53
    if '[INFO ][logstash.setting.writabledirectory] Creating directory {:setting=>"path.queue", :path=>"/var/lib/logstash/queue"}' in line:
        return 54
    if '[INFO ][logstash.setting.writabledirectory] Creating directory {:setting=>"path.dead_letter_queue", :path=>"/var/lib/logstash/dead_letter_queue"}' in line:
        return 55
    if '[INFO ][logstash.agent           ] No persistent UUID file found. Generating new UUID' in line:
        return 56
    if '[INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 429' in line:
        # an example of the full log line with variable 'data' looks like:
        # [INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 429 ({"type"=>"circuit_breaking_exception", "reason"=>"[parent] Data too large, data for [indices:data/write/bulk[s]] would be [4087188944/3.8gb], which is larger than the limit of [4063657984/3.7gb], real usage: [4087081784/3.8gb], new bytes reserved: [107160/104.6kb], usages [request=0/0b, fielddata=0/0b, in_flight_requests=107160/104.6kb, accounting=71938248/68.6mb]", "bytes_wanted"=>4087188944, "bytes_limit"=>4063657984, "durability"=>"PERMANENT"})
        return 57
    if '[INFO ][logstash.outputs.amazonelasticsearch] Retrying individual bulk actions that failed or were rejected by the previous bulk request. {:count=>' in line:
        return 58
    
    # additional seen in 'staging':
    if '[INFO ][logstash.inputs.s3       ] Registering {:bucket=>"ons-dp-staging-elb-logs", :region=>"eu-west-2"}' in line:
        return 59
    if '[INFO ][logstash.inputs.s3       ] Registering {:bucket=>"ons-dp-staging-flow-logs", :region=>"eu-west-2"}' in line:
        return 60
    if '[ERROR][logstash.agent           ] Failed to execute action {:action=>LogStash::PipelineAction::Stop/pipeline_id:main, :exception=>"Java::JavaLang::NullPointerException", :message=>"", :backtrace=>["org.logstash.common.io.DeadLetterQueueReader.getCurrentSegment' in line:
        return 61
    if "[FATAL][org.logstash.Logstash    ] Logstash stopped processing because of an error: (Error) Don't know how to handle `Java::JavaLang::NullPointerException` for `PipelineAction::Stop<main>`" in line:
        return 62
    if '[WARN ][io.netty.util.concurrent.SingleThreadEventExecutor] Unexpected exception from an event executor' in line:
        return 63
    if '[ERROR][logstash.inputs.s3       ] Unable to list objects in bucket {:exception=>Aws::S3::Errors::SlowDown, :message=>"Please reduce your request rate."' in line:
        return 64
    
    # additional seen in 'sandbox':
    if '[ERROR][org.logstash.common.io.DeadLetterQueueWriter] cannot write event to DLQ(path: /var/lib/logstash/dead_letter_queue/main): reached maxQueueSize of 1073741824' in line:
        return 65
    if '[INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 403 ({"type"=>"cluster_block_exception", "reason"=>"index [' in line:
        # an example of the full log line with variable 'data' looks like:
        # [INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 403 ({"type"=>"cluster_block_exception", "reason"=>"index [applogs-2023-01-09] blocked by: [FORBIDDEN/4/index preparing to close. Reopen the index to allow writes again or retry closing the index to fully close the index.];"})
        return 66
    if '[INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 503 ({"type"=>"unavailable_shards_exception", "reason"=>"[' in line:
        # an example of the full log line with variable 'data' looks like:
        # [INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 503 ({"type"=>"unavailable_shards_exception", "reason"=>"[auditlog-2023-01-09][0] primary shard is not active Timeout: [1m], request: [BulkShardRequest [[auditlog-2023-01-09][0]] containing [28] requests]"})
        return 67
    if '[INFO ][logstash.inputs.s3       ] Registering {:bucket=>"ons-dp-sandbox-elb-logs", :region=>"eu-west-2"}' in line:
        return 68
    if '[INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 403 ({"type"=>"index_create_block_exception", "reason"=>"blocked by: [FORBIDDEN' in line:
        # an example of the full log line with variable 'data' looks like:
        # [INFO ][logstash.outputs.amazonelasticsearch] retrying failed action with response code: 403 ({"type"=>"index_create_block_exception", "reason"=>"blocked by: [FORBIDDEN/10/cluster create-index blocked (api)];"})
        return 69
    if '[INFO ][logstash.inputs.s3       ] Registering {:bucket=>"ons-dp-sandbox-flow-logs", :region=>"eu-west-2"}' in line:
        return 70
    if '[ERROR][logstash.agent           ] Failed to execute action {:id=>:main, :action_type=>LogStash::ConvergeResult::FailedAction, :message=>"Expected one of' in line:
        # an example of the full log line with variable 'data' looks like:
        # [ERROR][logstash.agent           ] Failed to execute action {:id=>:main, :action_type=>LogStash::ConvergeResult::FailedAction, :message=>"Expected one of [ \\t\\r\\n], \"#\", [A-Za-z0-9_-], '\"', \"'\", [A-Za-z_], \"-\", [0-9], \"[\", \"{\" at line 10, column 80 (byte 262) after filter {\n  if [fields][type] == \"cantabular\" {\n    fingerprint {\n      source => [\"message\"]\n      target => \"[@metadata][fingerprint]\"\n      method => \"MURMUR3\"\n    }\n\n    mutate {\n      remove_field => [ \"agent\", \"offset\", \"host\", \"ecs\",\"port\",\"log\", \"input\",", :backtrace=>["/usr/share/logstash/logstash-core/lib/logstash/compiler.rb:32:in `compile_imperative'", "org/logstash/execution/AbstractPipelineExt.java:184:in `initialize'", "org/logstash/execution/JavaBasePipelineExt.java:69:in `initialize'", "/usr/share/logstash/logstash-core/lib/logstash/pipeline_action/reload.rb:53:in `execute'", "/usr/share/logstash/logstash-core/lib/logstash/agent.rb:371:in `block in converge_state'"]}
        return 71
    if "[ERROR][logstash.filters.mutate  ] Unknown setting 'match' for mutate" in line:
        return 72
    if '[ERROR][logstash.agent           ] Failed to execute action {:id=>:main, :action_type=>LogStash::ConvergeResult::FailedAction, :message=>"Unable to configure plugins: (ConfigurationError) Something is wrong with your configuration."' in line:
        # an example of the full log line with variable 'data' looks like:
        # [ERROR][logstash.agent           ] Failed to execute action {:id=>:main, :action_type=>LogStash::ConvergeResult::FailedAction, :message=>"Unable to configure plugins: (ConfigurationError) Something is wrong with your configuration.", :backtrace=>["org.logstash.config.ir.CompiledPipeline.<init>(CompiledPipeline.java:119)", "org.logstash.execution.JavaBasePipelineExt.initialize(JavaBasePipelineExt.java:83)", "org.logstash.execution.JavaBasePipelineExt$INVOKER$i$1$0$initialize.call(JavaBasePipelineExt$INVOKER$i$1$0$initialize.gen)", "org.jruby.internal.runtime.methods.JavaMethod$JavaMethodN.call(JavaMethod.java:837)", "org.jruby.runtime.callsite.CachingCallSite.cacheAndCall(CachingCallSite.java:332)", "org.jruby.runtime.callsite.CachingCallSite.call(CachingCallSite.java:86)", "org.jruby.RubyClass.newInstance(RubyClass.java:939)", "org.jruby.RubyClass$INVOKER$i$newInstance.call(RubyClass$INVOKER$i$newInstance.gen)", "org.jruby.ir.targets.InvokeSite.invoke(InvokeSite.java:207)", "usr.share.logstash.logstash_minus_core.lib.logstash.pipeline_action.reload.RUBY$method$execute$0(/usr/share/logstash/logstash-core/lib/logstash/pipeline_action/reload.rb:53)", "usr.share.logstash.logstash_minus_core.lib.logstash.pipeline_action.reload.RUBY$method$execute$0$__VARARGS__(/usr/share/logstash/logstash-core/lib/logstash/pipeline_action/reload.rb)", "org.jruby.internal.runtime.methods.CompiledIRMethod.call(CompiledIRMethod.java:80)", "org.jruby.internal.runtime.methods.MixedModeIRMethod.call(MixedModeIRMethod.java:70)", "org.jruby.ir.targets.InvokeSite.invoke(InvokeSite.java:207)", "usr.share.logstash.logstash_minus_core.lib.logstash.agent.RUBY$block$converge_state$2(/usr/share/logstash/logstash-core/lib/logstash/agent.rb:371)", "org.jruby.runtime.CompiledIRBlockBody.callDirect(CompiledIRBlockBody.java:138)", "org.jruby.runtime.IRBlockBody.call(IRBlockBody.java:58)", "org.jruby.runtime.IRBlockBody.call(IRBlockBody.java:52)", "org.jruby.runtime.Block.call(Block.java:139)", "org.jruby.RubyProc.call(RubyProc.java:318)", "org.jruby.internal.runtime.RubyRunnable.run(RubyRunnable.java:105)", "java.base/java.lang.Thread.run(Thread.java:834)"]}
        return 73
    if '[INFO ][org.logstash.beats.BeatsHandler] [local:' in line:
        # an example of the full log line with variable 'data' looks like:
        # [INFO ][org.logstash.beats.BeatsHandler] [local: 10.30.142.112:5044, remote: 10.30.142.234:42000] Handling exception: io.netty.handler.codec.DecoderException: org.logstash.beats.InvalidFrameProtocolException: Invalid version of beats protocol: 71 (caused by: org.logstash.beats.InvalidFrameProtocolException: Invalid version of beats protocol: 71)
        return 74

    # Do something with 'line'
    warn = "Unknown error line: " + line
    r_warn(warn)
    r_warn("You will need to add it to the above if's and add a new return code and process that accordingly wherever needed !")
    r_die(1, "Please fix the above problem")

class bcolors:
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKCYAN = '\033[96m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'

def r_die(return_code, error_message):
    print(f"{bcolors.FAIL}"+error_message+f"{bcolors.ENDC}")
    exit(return_code)

def r_warn(warn_message):
    print(f"{bcolors.WARNING}WARN  "+warn_message+f"{bcolors.ENDC}")

def r_info(info_message):
    print(info_message+f"{bcolors.ENDC}")

def r_trace(trace_message):
    print(f"{bcolors.OKBLUE}"+trace_message+f"{bcolors.ENDC}")

max_n = 0

print("Find the error counts on a daily basis\n")

# Now we work through each individual log file to build up a list of all days and counts of what
# error types happened that day, and accumulate data for a second pass

date_list = [] # a list of all the found dates in the logs, used to show results in date order

def is_date(string, fuzzy=False):
    """
    Return whether the string can be interpreted as a date.

    :param string: str, string to check for date
    :param fuzzy: bool, ignore unknown tokens in string if True
    """
    try: 
        parse(string, fuzzy=fuzzy)
        return True

    except ValueError:
        return False
    
def add_date(line):
    # The format of a valid log line starts like this
    #[2023-06-07T00:20:19,651]

    if line[0] != '[':
        return False, ""

    if is_date(line[1:20]):
        date = line[1:11]
        if date not in date_list:
            date_list.append(date)

        return True, date
    
    r_warn("not date: ", line) # Hmmm ?

    return False, ""

total_counts = {}   # a Dictionary of dates with the counts of each error on that day

def usage(additional_info):
    print("process-logs [<env>]")
    print("")
    print("optional argument:")
    en = ""
    for i, arg in enumerate(all_envs):
        if i == 0:
            en = arg
        else:
            en = en + " " + arg
    print("    <env>     is one of: ", en)
    print("")
    print("Optional argument: 'save' - to save all uncompressed log lines to files")
    print("")
    print("Process logstash logs that have been downloaded for the desired environment")
    if additional_info != "":
        print("")
        print(additional_info)
        exit(2)
    exit(1)

# Where to get the log files from (that have been downloaded by other script)
prod_dir_list = ["prod/logstash-1", "prod/logstash-2", "prod/logstash-3"]
sandbox_dir_list = ["sandbox/logstash-1", "sandbox/logstash-2", "sandbox/logstash-3"]
staging_dir_list = ["staging/logstash-1", "staging/logstash-2", "staging/logstash-3"]

# process command args
all_envs=["prod", "sandbox", "staging"]

envs = []
envs.append(all_envs[0]) # set default
envs_from_args=0

do_save = False

# parse args
for i, arg in enumerate(sys.argv):
    if i > 0:
        if arg == "-h" or arg == "--help":
            usage("")
            exit(1)

        if arg == 'save':
            do_save = True
            continue

        arg_ok = 0

        # check parameter matchs known environments, and if it does,
        #  take just the one environment to process
        for env in all_envs:
            if arg == env:
                envs=[]
                envs.append(arg)
                arg_ok=1
        if arg_ok == 1:
            continue

        usage("Unrecognised arg: "+arg)


dir_list = []

if envs[0] == "prod":
    dir_list = prod_dir_list
    r_info(f"{bcolors.BOLD}Processing: 'prod' logstash files\n")
elif envs[0] == "sandbox":
    dir_list = sandbox_dir_list
    r_info(f"{bcolors.BOLD}Processing: 'sandbox' logstash files\n")
elif envs[0] == "staging":
    dir_list = staging_dir_list
    r_info(f"{bcolors.BOLD}Processing: 'staging' logstash files\n")

running_error_counts = [0] * MAX_ERRORS

full_log_path = "full-logs"

if do_save:
    print("Uncompressed logs will be saved in:", full_log_path)
    if not os.path.exists(full_log_path):
        os.makedirs(full_log_path)

for dir in dir_list:
    d = f"{bcolors.OKCYAN}Directory: " + dir
    r_info(d)

    if do_save:
        result_name = dir.replace("/logstash", "") + ".txt"
        result_name = full_log_path + "/" + result_name
        result_file = open(result_name, 'w')

    for file in Path(dir).glob('*'):
        daily_error_counts = [0] * MAX_ERRORS

        if "logstash-slowlog-plain" in str(file):
            continue

        if "logstash-plain.log" in str(file):
            found_date = False
            date_within_file = ""
            with open(file) as f:
                for line in f:
                    if do_save:
                        result_file.write(line)
                    if "error" in line or "ERROR" in line or "WARN" in line or "INFO" in line:
                        if len(line) <= 2:
                            continue
                        if line[0] == " " and line[1] == " ":
                            if "error" not in line:
                                continue
                        e_num = process_line(line)
                        if e_num >= MAX_ERRORS:
                            # we should not get here, but just in case
                            r_die(3, "MAX_ERRORS more than: ", MAX_ERRORS, " so, increase it")
                        daily_error_counts[e_num] += 1
                        if found_date == False:
                            res, date_within_file = add_date(line)
                            if res == True:
                                found_date = True # we have found a date for this file
                        if e_num > max_n:
                            max_n = e_num
            
            daily_list = []
            if date_within_file != "":
                for i in range(MAX_ERRORS):
                    running_error_counts[i] += daily_error_counts[i] # accumulate totals for check later to catch any code errors
                    daily_list.append(daily_error_counts[i])

                if date_within_file in total_counts:
                    # add data from this logstash box to date from other box(es)
                    existing_list = total_counts[date_within_file]
                    for i in range(MAX_ERRORS):
                        daily_list[i] += existing_list[i]

                total_counts[date_within_file] = daily_list
            else:
                r_info("No date found in file: ", file) # Hmm ?

        if ".log.gz" in str(file):
            found_date = False
            date_within_file = ""
            with gzip.open(file, 'rt') as f:
                for line in f:
                    if do_save:
                        result_file.write(line)
                    if "error" in line or "ERROR" in line or "WARN" in line or "INFO" in line:
                        if len(line) <= 2:
                            continue
                        if line[0] == " " and line[1] == " ":
                            if "error" not in line:
                                continue
                        e_num = process_line(line)
                        if e_num >= MAX_ERRORS:
                            # we should not get here, but just in case
                            r_die(3, "MAX_ERRORS more than: ", MAX_ERRORS, " so, increase it")
                        daily_error_counts[e_num] += 1
                        if found_date == False:
                            res, date_within_file = add_date(line)
                            if res == True:
                                found_date = True # we have found a date for this file
                        if e_num > max_n:
                            max_n = e_num
            
            daily_list = []
            if date_within_file != "":
                for i in range(MAX_ERRORS):
                    running_error_counts[i] += daily_error_counts[i] # accumulate totals for check later to catch any code errors
                    daily_list.append(daily_error_counts[i])

                if date_within_file in total_counts:
                    # add data from this logstash box to date from other box(es)
                    existing_list = total_counts[date_within_file]
                    for i in range(MAX_ERRORS):
                        daily_list[i] += existing_list[i]

                total_counts[date_within_file] = daily_list
            else:
                r_info("No date found in file: ", file) # Hmm ?

    if do_save:
        result_file.close()



# now add up the total_counts to check they match the running_error_counts
check_error_counts = [0] * MAX_ERRORS
for key, value in total_counts.items():
    for i in range(max_n+1):
        check_error_counts[i] += value[i]

print("\nrunning_error_counts: 'error number' and its 'total count'\n")

for i in range(max_n+1):
    print("running_error_counts:", i, "is:", running_error_counts[i])
    if running_error_counts[i] != check_error_counts[i]:
        r_die(3, "Error: check_error_counts is different :", i, "is:", check_error_counts[i])

# Now that we have the files checked over and know the limits, we can report the results for plotting
date_list.sort()

print("number of dates: ", len(date_list))


# determine max widths of printed numbers for each count - to print rows with same spacing
max_total_counts_widths = [0] * MAX_ERRORS

for date in date_list:
    counts = total_counts[date]
    for i in range(max_n+1):
        s = str(counts[i])
        l = len(s)
        if l > max_total_counts_widths[i]:
            max_total_counts_widths[i] = l

# ensure a minimum column width to space numbers out correctly below column headings
for i in range(max_n+1):
    if max_total_counts_widths[i] < 2:
        max_total_counts_widths[i] = 2


# create and print header row of error numbers
err_nums = "          Error numbers: ["
for i in range(max_n+1):
    s2 = str(i)
    l = len(s2)
    while l < max_total_counts_widths[i]:
        err_nums += " "
        l += 1
    err_nums += s2 + " "
err_nums += "]"
r_trace(err_nums)


previous_error_counts = [0] * MAX_ERRORS

# now print counts by column width in 'date' order
for date in date_list:
    d = parse(date).strftime("%a")
    if d == "Mon":
        r_trace(err_nums)   # show the row column header of the error numbers before every Monday

    counts = total_counts[date]
    day_counts = "Date: " + str(date) + " Counts: ["
    for i in range(max_n+1):
        s2 = str(counts[i])
        l = len(s2)
        while l < max_total_counts_widths[i]:
            day_counts += " "
            l += 1
        if s2 == "0":
            s2 = " "    # don't show zero's, so that actual numbers stand out
        if counts[i] > previous_error_counts[i]:
            s2 = f"{bcolors.WARNING}"+s2+f"{bcolors.ENDC}"  # Yellow count is going up
        elif counts[i] < previous_error_counts[i]:
            s2 = f"{bcolors.OKGREEN}"+s2+f"{bcolors.ENDC}"  # Green  count is going down

        previous_error_counts[i] = counts[i]
        day_counts += s2 + " "
    day_counts += "]"
    print(day_counts)
