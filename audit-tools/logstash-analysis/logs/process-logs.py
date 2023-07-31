#!/usr/bin/python3

import sys
import os
from pathlib import Path
import gzip
from dateutil.parser import parse
from datetime import date, timedelta

MAX_ERRORS = 200

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
        # for example:
        # Unable to download remote file {:exception=>Aws::S3::Errors::NoSuchKey, :message=>"The specified key does not exist.", :remote_key=>"AWSLogs/337289154253/elasticloadbalancing/eu-west-2/2023/07/26/337289154253_elasticloadbalancing_eu-west-2_app.predissemination-sandbox.33e29f9618383559_20230726T1215Z_10.30.142.230_1qza4pxa.log.gz"}
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
    
    # DEBUG events:
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"event":"failed to request service health"' in line:
        # an example of the full log line with variable 'data' looks like:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-63 >> "{"event":"failed to request service health","namespace":"dp-frontend-release-calendar","@version":"1","created_at":"2023-07-07T14:48:35.489943253Z","severity":1,"errors":[{"message":"invalid response from downstream service - should be: 200, got: 503, path: /health","stack_trace":[{"file":"/go/pkg/mod/github.com/!o!n!sdigital/log.go/v2@v2.4.1/log/log.go","line":93,"function":"github.com/ONSdigital/log.go/v2/log.Error"},{"file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-api-clients-go/v2@v2.252.1/health/health.go","line":90,"function":"github.com/ONSdigital/dp-api-clients-go/v2/health.(*Client).Checker"},{"file":"/tmp/build/80754af9/dp-frontend-release-calendar/handlers/babbage.go","line":42,"function":"github.com/ONSdigital/dp-frontend-release-calendar/handlers.(*BabbageClient).Checker"},{"file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-healthcheck@v1.6.1/healthcheck/ticker.go","line":56,"function":"github.com/ONSdigital/dp-healthcheck/healthcheck.(*ticker).runCheck"},{"file":"/usr/local/go/src/runtime/asm_amd64.s","line":1598,"function":"runtime.goexit"}]}],"tags":[],"@timestamp":"2023-07-07T14:48:35.489Z","nomad":{"allocation_id":"e461d54f-493a-1c23-d2f1-b45bf2d937c1","host":"ip-10-30-141-96"},"data":"{\n  \"service\": \"Babbage\"\n}"}[\n]"
        return 75
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"took' in line:
        # an example of the full log line with variable 'data' looks like:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-12 << "{"took":5,"errors":false,"items":[{"index":{"_index":"nginxlog-2023-07-07","_type":"_doc","_id":"1149708251","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":1504271,"_primary_term":1,"status":201}}]}"
        return 76
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_timestamp":"' in line:
        # an example of the full log line with variable 'data' looks like:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-11 >> "{"syslog_timestamp":"Jul  7 14:48:44","message":"    2023-07-07T14:48:44.652Z [WARN]  agent: Check is now critical: check=_nomad-check-151e0f20638b39d046a4fa408a63ddf4ad8d2405","syslog_hostname":"ip-10-30-140-12","syslog_program":"consul","@version":"1","received_at":"2023-07-07T14:48:44.789Z","ecs":{"version":"1.6.0"},"agent":{"name":"ip-10-30-140-12","ephemeral_id":"b5815037-0252-4b49-831a-4cc67f6d4c8d","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","version":"7.11.2","hostname":"ip-10-30-140-12","type":"filebeat"},"received_from":"{\"name\":\"ip-10-30-140-12\"}","fields":{"type":"system"},"syslog_pid":"31572","syslog_severity":"notice","syslog_facility":"user-level","tags":["beats_input_codec_plain_applied"],"syslog_facility_code":1,"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:44.000Z","log":{"offset":30346631,"file":{"path":"/var/log/syslog"}},"syslog_severity_code":5}[\n]"
        return 77
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":"' in line:
        # an example of the full log line with variable 'data' looks like:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-80 >> "{"message":"[2023-07-07T13:56:10,069][DEBUG][org.apache.http.wire     ] http-outgoing-13 >> \"{\"syslog_timestamp\":\"Jul  7 13:56:08\",\"message\":\"2023/07/07 13:56:08.369450 [INFO] (runner) executing command \\\"/bin/systemctl reload nginx\\\" from \\\"/var/lib/consul-template/web-nginx.http.conf.tpl\\\" => \\\"/etc/nginx/conf.d/web-nginx.http.conf\\\"\",\"syslog_hostname\":\"ip-10-30-140-12\",\"syslog_program\":\"consul-template\",\"@version\":\"1\",\"received_at\":\"2023-07-07T13:56:09.022Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-140-12\",\"ephemeral_id\":\"b5815037-0252-4b49-831a-4cc67f6d4c8d\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-140-12\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-140-12\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"3362013\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:56:08.000Z\",\"log\":{\"offset\":28570219,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"@version":"1","ecs":{"version":"1.7.0"},"agent":{"name":"ip-10-30-142-35","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","hostname":"ip-10-30-142-35","type":"filebeat","version":"7.11.2"},"fileset":{"name":"log"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:44.557Z","host":{"name":"ip-10-30-142-35"},"log":{"offset":35112287,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}[\n]"
        return 78
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"event":' in line:
        # an example of the full log line with variable 'data' looks like:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-58 >> "{"event":"error from graph DB","namespace":"./dp-observation-api","@version":"1","created_at":"2023-07-07T14:48:43.582903651Z","severity":1,"error":{"message":"write tcp 172.17.0.22:53744->10.30.136.184:8182: write: broken pipe","stack_trace":[{"file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.13.1/graph/error_consumer.go","line":20,"function":"github.com/ONSdigital/dp-graph/v2/graph.NewLoggingErrorConsumer.func1"},{"file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.13.1/graph/error_consumer.go","line":38,"function":"github.com/ONSdigital/dp-graph/v2/graph.NewErrorConsumer.func1"},{"file":"/usr/local/go/src/runtime/asm_amd64.s","line":1594,"function":"runtime.goexit"}],"data":"{\n\n}"},"tags":[],"@timestamp":"2023-07-07T14:48:43.582Z","nomad":{"allocation_id":"61de301e-a365-ee97-d66c-fd242b5e888c","host":"ip-10-30-138-234"}}[\n]"
        return 79
    if '[DEBUG][org.apache.http.conn.ssl.SSLConnectionSocketFactory] Enabled cipher suites:' in line:
        # an example of the full log line with variable 'data' looks like:
        # [DEBUG][org.apache.http.conn.ssl.SSLConnectionSocketFactory] Enabled cipher suites:[TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384, TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, TLS_RSA_WITH_AES_256_GCM_SHA384, TLS_ECDH_ECDSA_WITH_AES_256_GCM_SHA384, TLS_ECDH_RSA_WITH_AES_256_GCM_SHA384, TLS_DHE_RSA_WITH_AES_256_GCM_SHA384, TLS_DHE_DSS_WITH_AES_256_GCM_SHA384, TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, TLS_RSA_WITH_AES_128_GCM_SHA256, TLS_ECDH_ECDSA_WITH_AES_128_GCM_SHA256, TLS_ECDH_RSA_WITH_AES_128_GCM_SHA256, TLS_DHE_RSA_WITH_AES_128_GCM_SHA256, TLS_DHE_DSS_WITH_AES_128_GCM_SHA256, TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384, TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384, TLS_RSA_WITH_AES_256_CBC_SHA256, TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA384, TLS_ECDH_RSA_WITH_AES_256_CBC_SHA384, TLS_DHE_RSA_WITH_AES_256_CBC_SHA256, TLS_DHE_DSS_WITH_AES_256_CBC_SHA256, TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, TLS_RSA_WITH_AES_256_CBC_SHA, TLS_ECDH_ECDSA_WITH_AES_256_CBC_SHA, TLS_ECDH_RSA_WITH_AES_256_CBC_SHA, TLS_DHE_RSA_WITH_AES_256_CBC_SHA, TLS_DHE_DSS_WITH_AES_256_CBC_SHA, TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, TLS_RSA_WITH_AES_128_CBC_SHA256, TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA256, TLS_ECDH_RSA_WITH_AES_128_CBC_SHA256, TLS_DHE_RSA_WITH_AES_128_CBC_SHA256, TLS_DHE_DSS_WITH_AES_128_CBC_SHA256, TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, TLS_RSA_WITH_AES_128_CBC_SHA, TLS_ECDH_ECDSA_WITH_AES_128_CBC_SHA, TLS_ECDH_RSA_WITH_AES_128_CBC_SHA, TLS_DHE_RSA_WITH_AES_128_CBC_SHA, TLS_DHE_DSS_WITH_AES_128_CBC_SHA, TLS_EMPTY_RENEGOTIATION_INFO_SCSV]
        return 80
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in  line and '"[read] I/O error: Read timed out' in line:
        # this copes with all variants whereby the number after 'outgoing-' varies
        # an example of the full log line with variable 'data' looks like:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-3 << "[read] I/O error: Read timed out"
        return 81
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'self._raise_error(response.status, raw_data)' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-56 >> "{"message":"    self._raise_error(response.status, raw_data)","@version":"1","fields":{"type":"application"},"tags":[],"@timestamp":"2023-07-07T14:48:14.613Z","nomad":{"allocation_id":"c5a93b20-2f60-d654-013e-a55b2fb7e644","host":"ip-10-30-138-203"}}[\n]"
        return 82
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":"  File ' in line and '"/usr/local/lib/python3.9/site-packages/elasticsearch2/connection/base.py' in line and '", line 114, in _raise_error","@version":' in line and '"nomad":{"allocation_id":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-56 >> "{"message":"  File \"/usr/local/lib/python3.9/site-packages/elasticsearch2/connection/base.py\", line 114, in _raise_error","@version":"1","fields":{"type":"application"},"tags":[],"@timestamp":"2023-07-07T14:48:14.613Z","nomad":{"allocation_id":"c5a93b20-2f60-d654-013e-a55b2fb7e644","host":"ip-10-30-138-203"}}[\n]"
        return 83
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'raise HTTP_EXCEPTIONS.get(status_code, TransportError)(status_code, error_message, additional_info)' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-56 >> "{"message":"    raise HTTP_EXCEPTIONS.get(status_code, TransportError)(status_code, error_message, additional_info)","@version":"1","fields":{"type":"application"},"tags":[],"@timestamp":"2023-07-07T14:48:14.613Z","nomad":{"allocation_id":"c5a93b20-2f60-d654-013e-a55b2fb7e644","host":"ip-10-30-138-203"}}[\n]"
        return 84
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":' in line and '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  agent: Synced check: check=_nomad-check' in line and '"filebeat' in line and 'beats_input_codec_plain_applied' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-73 >> "{"message":"[2023-07-07T14:02:29,187][DEBUG][org.apache.http.wire     ] http-outgoing-18 >> \"{\"syslog_timestamp\":\"Jul  7 14:02:28\",\"message\":\"    2023-07-07T14:02:28.048Z [INFO]  agent: Synced check: check=_nomad-check-4d9d26911ef6b613b2b0ee61031aca08453b2cbc\",\"syslog_hostname\":\"ip-10-30-141-96\",\"syslog_program\":\"consul\",\"@version\":\"1\",\"received_at\":\"2023-07-07T14:02:28.115Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-141-96\",\"ephemeral_id\":\"c480881b-1b2f-4ed6-ab95-61985140fb53\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-141-96\",\"type\":\"filebeat\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-141-96\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"1116313\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:02:28.000Z\",\"log\":{\"offset\":82808623,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"@version":"1","ecs":{"version":"1.7.0"},"agent":{"name":"ip-10-30-142-35","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"fileset":{"name":"log"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:16.396Z","host":{"name":"ip-10-30-142-35"},"log":{"offset":29942517,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}[\n]"
        return 85
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":' in line and '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"syslog_timestamp' in line and '[INFO] (runner) rendered' in line and '/var/lib/consul-template/publishing-nginx.http.conf.tpl' in line and '"/etc/nginx/conf.d/publishing-nginx.http.conf' in line and '"consul-template' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-73 >> "{"message":"[2023-07-07T14:45:27,316][DEBUG][org.apache.http.wire     ] http-outgoing-73 >> \"{\"message\":\"[2023-07-07T14:28:12,360][DEBUG][org.apache.http.wire     ] http-outgoing-18 >> \\\"{\\\"syslog_timestamp\\\":\\\"Jul  7 14:28:10\\\",\\\"message\\\":\\\"2023/07/07 14:28:10.057715 [INFO] (runner) rendered \\\\\\\"/var/lib/consul-template/publishing-nginx.http.conf.tpl\\\\\\\" => \\\\\\\"/etc/nginx/conf.d/publishing-nginx.http.conf\\\\\\\"\\\",\\\"syslog_hostname\\\":\\\"ip-10-30-138-41\\\",\\\"syslog_program\\\":\\\"consul-template\\\",\\\"@version\\\":\\\"1\\\",\\\"received_at\\\":\\\"2023-07-07T14:28:11.058Z\\\",\\\"ecs\\\":{\\\"version\\\":\\\"1.6.0\\\"},\\\"agent\\\":{\\\"name\\\":\\\"ip-10-30-138-41\\\",\\\"ephemeral_id\\\":\\\"87f5ab17-ed70-4e4e-b649-6921642da187\\\",\\\"id\\\":\\\"c81ff488-13bd-4915-83df-f9dd60fa5444\\\",\\\"type\\\":\\\"filebeat\\\",\\\"version\\\":\\\"7.11.2\\\",\\\"hostname\\\":\\\"ip-10-30-138-41\\\"},\\\"received_from\\\":\\\"{\\\\\\\"name\\\\\\\":\\\\\\\"ip-10-30-138-41\\\\\\\"}\\\",\\\"fields\\\":{\\\"type\\\":\\\"system\\\"},\\\"syslog_pid\\\":\\\"11295\\\",\\\"syslog_severity\\\":\\\"notice\\\",\\\"syslog_facility\\\":\\\"user-level\\\",\\\"tags\\\":[\\\"beats_input_codec_plain_applied\\\"],\\\"syslog_facility_code\\\":1,\\\"input\\\":{\\\"type\\\":\\\"log\\\"},\\\"@timestamp\\\":\\\"2023-07-07T14:28:10.000Z\\\",\\\"log\\\":{\\\"offset\\\":44301621,\\\"file\\\":{\\\"path\\\":\\\"/var/log/syslog\\\"}},\\\"syslog_severity_code\\\":5}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:45:26.491Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":9033103,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\"","event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"@version":"1","ecs":{"version":"1.7.0"},"agent":{"name":"ip-10-30-142-35","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"fileset":{"name":"log"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:19.940Z","host":{"name":"ip-10-30-142-35"},"log":{"offset":1045632,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}[\n]"
        return 86
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'syslog_timestamp' in line and 'INFO [monitoring] log/log.go:184 Non-zero metrics in the last 30s' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-73 >> "{"message":"[2023-07-07T14:38:20,458][DEBUG][org.apache.http.wire     ] http-outgoing-18 >> \"{\"syslog_timestamp\":\"Jul  7 14:38:19\",\"message\":\"INFO [monitoring] log/log.go:184 Non-zero metrics in the last 30s {\\\"monitoring\\\": {\\\"metrics\\\": {\\\"beat\\\":{\\\"cgroup\\\":{\\\"cpuacct\\\":{\\\"total\\\":{\\\"ns\\\":26397564}},\\\"memory\\\":{\\\"mem\\\":{\\\"usage\\\":{\\\"bytes\\\":102400}}}},\\\"cpu\\\":{\\\"system\\\":{\\\"ticks\\\":1866370,\\\"time\\\":{\\\"ms\\\":2}},\\\"total\\\":{\\\"ticks\\\":8411240,\\\"time\\\":{\\\"ms\\\":26},\\\"value\\\":8411240},\\\"user\\\":{\\\"ticks\\\":6544870,\\\"time\\\":{\\\"ms\\\":24}}},\\\"handles\\\":{\\\"limit\\\":{\\\"hard\\\":524288,\\\"soft\\\":1024},\\\"open\\\":21},\\\"info\\\":{\\\"ephemeral_id\\\":\\\"5d2fb355-e702-4419-9727-148b59ed72bc\\\",\\\"uptime\\\":{\\\"ms\\\":8817900086},\\\"version\\\":\\\"7.17.7\\\"},\\\"memstats\\\":{\\\"gc_next\\\":23538504,\\\"memory_alloc\\\":17410600,\\\"memory_total\\\":3342649435896,\\\"rss\\\":110481408},\\\"runtime\\\":{\\\"goroutines\\\":98}},\\\"filebeat\\\":{\\\"events\\\":{\\\"active\\\":1,\\\"added\\\":19,\\\"done\\\":18},\\\"harvester\\\":{\\\"open_files\\\":4,\\\"running\\\":4}},\\\"libbeat\\\":{\\\"config\\\":{\\\"module\\\":{\\\"running\\\":7}},\\\"output\\\":{\\\"events\\\":{\\\"acked\\\":18,\\\"active\\\":0,\\\"batches\\\":7,\\\"total\\\":18},\\\"read\\\":{\\\"bytes\\\":42},\\\"write\\\":{\\\"bytes\\\":4933}},\\\"pipeline\\\":{\\\"clients\\\":7,\\\"events\\\":{\\\"active\\\":5,\\\"published\\\":19,\\\"total\\\":19},\\\"queue\\\":{\\\"acked\\\":18}}},\\\"registrar\\\":{\\\"states\\\":{\\\"current\\\":6,\\\"update\\\":18},\\\"writes\\\":{\\\"success\\\":11,\\\"total\\\":11}},\\\"system\\\":{\\\"load\\\":{\\\"1\\\":0,\\\"15\\\":0,\\\"5\\\":0.01,\\\"norm\\\":{\\\"1\\\":0,\\\"15\\\":0,\\\"5\\\":0.0025}}}}}}\",\"syslog_hostname\":\"ip-10-30-137-71\",\"syslog_program\":\"filebeat\",\"@version\":\"1\",\"received_at\":\"2023-07-07T14:38:19.839Z\",\"ecs\":{\"version\":\"1.12.0\"},\"agent\":{\"name\":\"ip-10-30-137-71\",\"ephemeral_id\":\"5d2fb355-e702-4419-9727-148b59ed72bc\",\"id\":\"a1437f6b-3a75-4735-8e10-b8d93f5fbc9d\",\"type\":\"filebeat\",\"version\":\"7.17.7\",\"hostname\":\"ip-10-30-137-71\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-137-71\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"5000\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:38:19.000Z\",\"log\":{\"offset\":2157740,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"@version":"1","ecs":{"version":"1.7.0"},"agent":{"name":"ip-10-30-142-35","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"fileset":{"name":"log"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:20.625Z","host":{"name":"ip-10-30-142-35"},"log":{"offset":5179546,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}[\n]"
        return 87
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '>> "{"message"' in line and '"took' in line and '"applogs-' in line and '"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"}' in line and '"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-73 >> "{"message":"[2023-07-07T13:50:14,029][DEBUG][org.apache.http.wire     ] http-outgoing-1 << \"{\"took\":4,\"errors\":false,\"items\":[{\"index\":{\"_index\":\"applogs-2023-07-07\",\"_type\":\"_doc\",\"_id\":\"2973748310\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":2,\"successful\":2,\"failed\":0},\"_seq_no\":5914181,\"_primary_term\":1,\"status\":201}}]}\"","event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"@version":"1","ecs":{"version":"1.7.0"},"agent":{"name":"ip-10-30-142-35","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"fileset":{"name":"log"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:23.408Z","host":{"name":"ip-10-30-142-35"},"log":{"offset":43505371,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}[\n]"
        return 88
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'cantabularlogs-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-73 >> "{"message":"[2023-07-07T14:33:23,099][DEBUG][org.apache.http.wire     ] http-outgoing-57 << \"{\"took\":4,\"errors\":false,\"items\":[{\"index\":{\"_index\":\"cantabularlogs-2023-07-07\",\"_type\":\"_doc\",\"_id\":\"3520397353\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":2,\"successful\":2,\"failed\":0},\"_seq_no\":224987,\"_primary_term\":1,\"status\":201}}]}\"","event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"@version":"1","ecs":{"version":"1.7.0"},"agent":{"name":"ip-10-30-142-35","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"fileset":{"name":"log"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:25.804Z","host":{"name":"ip-10-30-142-35"},"log":{"offset":7590587,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}[\n]"
        return 89
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"event' in line and '"Ping: failed to retrieve response from http client' in line and '"namespace' in line and '"dp-apipoc-server' in line and 'dial tcp: lookup www.ons.gov.uk on 10.30.136.2:53: no such host' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-73 >> "{"message":"[2023-07-07T14:37:49,092][DEBUG][org.apache.http.wire     ] http-outgoing-73 >> \"{\"message\":\"[2023-07-07T13:38:53,294][DEBUG][org.apache.http.wire     ] http-outgoing-1 >> \\\"{\\\"event\\\":\\\"Ping: failed to retrieve response from http client\\\",\\\"namespace\\\":\\\"dp-apipoc-server\\\",\\\"@version\\\":\\\"1\\\",\\\"created_at\\\":\\\"2023-03-02T14:00:35.845944593Z\\\",\\\"severity\\\":1,\\\"errors\\\":[{\\\"message\\\":\\\"Get \\\\\\\"https://www.ons.gov.uk\\\\\\\": dial tcp: lookup www.ons.gov.uk on 10.30.136.2:53: no such host\\\",\\\"stack_trace\\\":[{\\\"file\\\":\\\"/go/pkg/mod/github.com/!o!n!sdigital/log.go/v2@v2.0.9/log/log.go\\\",\\\"line\\\":92,\\\"function\\\":\\\"github.com/ONSdigital/log.go/v2/log.Error\\\"},{\\\"file\\\":\\\"/tmp/build/80754af9/dp-apipoc-server/upstream/httpclient.go\\\",\\\"line\\\":42,\\\"function\\\":\\\"github.com/ONSdigital/dp-apipoc-server/upstream.(*httpService).Ping\\\"},{\\\"file\\\":\\\"/tmp/build/80754af9/dp-apipoc-server/upstream/website.go\\\",\\\"line\\\":26,\\\"function\\\":\\\"github.com/ONSdigital/dp-apipoc-server/upstream.(*websiteService).Ping\\\"},{\\\"file\\\":\\\"/tmp/build/80754af9/dp-apipoc-server/handler/ops.go\\\",\\\"line\\\":30,\\\"function\\\":\\\"github.com/ONSdigital/dp-apipoc-server/handler.OpsHandler.StatusHandler\\\"},{\\\"file\\\":\\\"/usr/local/go/src/net/http/server.go\\\",\\\"line\\\":2109,\\\"function\\\":\\\"net/http.HandlerFunc.ServeHTTP\\\"},{\\\"file\\\":\\\"/go/pkg/mod/github.com/gorilla/mux@v1.8.0/mux.go\\\",\\\"line\\\":210,\\\"function\\\":\\\"github.com/gorilla/mux.(*Router).ServeHTTP\\\"},{\\\"file\\\":\\\"/go/pkg/mod/github.com/rs/cors@v1.8.0/cors.go\\\",\\\"line\\\":219,\\\"function\\\":\\\"github.com/rs/cors.(*Cors).Handler.func1\\\"},{\\\"file\\\":\\\"/usr/local/go/src/net/http/server.go\\\",\\\"line\\\":2109,\\\"function\\\":\\\"net/http.HandlerFunc.ServeHTTP\\\"},{\\\"file\\\":\\\"/tmp/build/80754af9/dp-apipoc-server/routing/routes.go\\\",\\\"line\\\":61,\\\"function\\\":\\\"github.com/ONSdigital/dp-apipoc-server/routing.DeprecationMiddleware.func1.1\\\"},{\\\"file\\\":\\\"/usr/local/go/src/net/http/server.go\\\",\\\"line\\\":2109,\\\"function\\\":\\\"net/http.HandlerFunc.ServeHTTP\\\"}]}],\\\"tags\\\":[],\\\"@timestamp\\\":\\\"2023-03-02T14:00:35.845Z\\\",\\\"nomad\\\":{\\\"allocation_id\\\":\\\"7907d1cb-c24d-16df-2323-edbf9cd32fa0\\\",\\\"host\\\":\\\"ip-10-30-140-223\\\"}}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:37:48.612Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":28426917,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\"","event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"@version":"1","ecs":{"version":"1.7.0"},"agent":{"name":"ip-10-30-142-35","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"fileset":{"name":"log"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"@timestamp":"2023-07-07T14:48:27.872Z","host":{"name":"ip-10-30-142-35"},"log":{"offset":5636227,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"service":{"type":"logstash"}}[\n]"
        return 90
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '<< "{"took":' in line and ',"errors":false,"items":[{"index":{"_index":"nginxlog-' in line and '"result":"created","_shards":{"total":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-9 << "{"took":5,"errors":false,"items":[{"index":{"_index":"nginxlog-2023-07-07","_type":"_doc","_id":"3186269914","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":1510019,"_primary_term":1,"status":201}}]}"
        return 91
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '<< "{"took":' in line and ',"errors":false,"items":[{"index":{"_index":"authlog-' in line and '"result":"created","_shards":{"total":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-75 << "{"took":5,"errors":false,"items":[{"index":{"_index":"authlog-2023-07-07","_type":"_doc","_id":"2979270552","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":46875,"_primary_term":1,"status":201}}]}"
        return 92
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '<< "{"took":' in line and ',"errors":false,"items":[{"index":{"_index":"authlog-' in line and '"result":"updated","_shards":{"total":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-78 << "{"took":7,"errors":false,"items":[{"index":{"_index":"authlog-2023-07-07","_type":"_doc","_id":"1639827875","_version":3,"result":"updated","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":47271,"_primary_term":1,"status":200}}]}"
        return 93
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '<< "{"took":' in line and ',"errors":false,"items":[{"index":{"_index":"cantabularlogs-' in line and '"result":"created","_shards":{"total":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-48 << "{"took":6,"errors":false,"items":[{"index":{"_index":"cantabularlogs-2023-07-07","_type":"_doc","_id":"3654845773","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":228742,"_primary_term":1,"status":201}}]}"
        return 94
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '<< "{"took":' in line and ',"errors":false,"items":[{"index":{"_index":"applogs-' in line and '"result":"created","_shards":{"total":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-49 << "{"took":4,"errors":false,"items":[{"index":{"_index":"applogs-2023-07-07","_type":"_doc","_id":"4091735265","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":6231814,"_primary_term":1,"status":201}}]}"
        return 95
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":' in line and '[WARN] (view) health.service(elasticsearch-tcp|passing): Get https://127.0.0.1:8500/v1/health/service/elasticsearch-tcp?index=' in line and 'remote error: tls: bad certificate (retry attempt ' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:03.548343 [WARN] (view) health.service(elasticsearch-tcp|passing): Get https://127.0.0.1:8500/v1/health/service/elasticsearch-tcp?index=42564508&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 784125 after \"20s\")","@version":"1","@timestamp":"2023-07-07T14:48:03.000Z","syslog_timestamp":"Jul  7 14:48:03","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":72808652},"syslog_severity":"notice","received_at":"2023-07-07T14:48:03.772Z","agent":{"hostname":"ip-10-30-138-203","type":"filebeat","name":"ip-10-30-138-203","version":"7.11.2","ephemeral_id":"839041a5-29e2-4e7a-8063-efa3f5ee509e","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-203\"}","syslog_pid":"3838102","syslog_hostname":"ip-10-30-138-203","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 96
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":"(view) health.service(elasticsearch-tcp|passing): Get https://127.0.0.1:8500/v1/health/service/elasticsearch-tcp?index=' in line and 'remote error: tls: bad certificate (retry attempt' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"(view) health.service(elasticsearch-tcp|passing): Get https://127.0.0.1:8500/v1/health/service/elasticsearch-tcp?index=42564508&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 784125 after \"20s\")","@version":"1","@timestamp":"2023-07-07T14:48:03.000Z","syslog_timestamp":"Jul  7 14:48:03","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":72808974},"syslog_severity":"notice","received_at":"2023-07-07T14:48:03.772Z","agent":{"hostname":"ip-10-30-138-203","type":"filebeat","name":"ip-10-30-138-203","version":"7.11.2","ephemeral_id":"839041a5-29e2-4e7a-8063-efa3f5ee509e","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-203\"}","syslog_pid":"3838102","syslog_hostname":"ip-10-30-138-203","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 97
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' << "{"took":' in line and ',"errors":false,"items":[{"index":{"_index":"auditlog-' in line and '"result":"created","_shards":{"total":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-51 << "{"took":4,"errors":false,"items":[{"index":{"_index":"auditlog-2023-07-07","_type":"_doc","_id":"596521038","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":27285170,"_primary_term":1,"status":201}}]}"
        return 98
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message"' in line and '[WARN] (view) health.service(dp-charts-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-charts-api?index=' in line and 'remote error: tls: bad certificate (retry attempt ' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:03.220512 [WARN] (view) health.service(dp-charts-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-charts-api?index=42564487&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 784065 after \"20s\")","@version":"1","@timestamp":"2023-07-07T14:48:03.000Z","syslog_timestamp":"Jul  7 14:48:03","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51205998},"syslog_severity":"notice","received_at":"2023-07-07T14:48:04.072Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 99
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":"(view) health.service(dp-charts-api|passing):' in line and 'remote error: tls: bad certificate (retry attempt' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"(view) health.service(dp-charts-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-charts-api?index=42564487&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 784065 after \"20s\")","@version":"1","@timestamp":"2023-07-07T14:48:03.000Z","syslog_timestamp":"Jul  7 14:48:03","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51206310},"syslog_severity":"notice","received_at":"2023-07-07T14:48:04.072Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 100
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message"' in line and '[WARN] (view) health.service(dp-bulletin-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-bulletin-api?' in line and 'remote error: tls: bad certificate (retry attempt' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:03.657402 [WARN] (view) health.service(dp-bulletin-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-bulletin-api?index=42564487&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 784065 after \"20s\")","@version":"1","@timestamp":"2023-07-07T14:48:03.000Z","syslog_timestamp":"Jul  7 14:48:03","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51207178},"syslog_severity":"notice","received_at":"2023-07-07T14:48:04.072Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 101
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":"(view) health.service(dp-bulletin-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-bulletin-api' in line and 'remote error: tls: bad certificate (retry attempt' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"(view) health.service(dp-bulletin-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-bulletin-api?index=42564487&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 784065 after \"20s\")","@version":"1","@timestamp":"2023-07-07T14:48:03.000Z","syslog_timestamp":"Jul  7 14:48:03","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51207494},"syslog_severity":"notice","received_at":"2023-07-07T14:48:04.072Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 102
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' << "{"took":5,"errors":false,"items":[{"index":{"_index":"syslog-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 << "{"took":5,"errors":false,"items":[{"index":{"_index":"syslog-2023-07-07","_type":"_doc","_id":"1773097415","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":2323941,"_primary_term":1,"status":201}}]}"
        return 103
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO] (runner) executing command' in line and '/var/lib/consul-template/publishing-nginx.http.conf.tpl' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:04.546183 [INFO] (runner) executing command \"/bin/systemctl reload nginx\" from \"/var/lib/consul-template/publishing-nginx.http.conf.tpl\" => \"/etc/nginx/conf.d/publishing-nginx.http.conf\"","@version":"1","@timestamp":"2023-07-07T14:48:04.000Z","syslog_timestamp":"Jul  7 14:48:04","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51208857},"syslog_severity":"notice","received_at":"2023-07-07T14:48:05.072Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 104
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO] (child) spawning: /bin/systemctl reload nginx' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:04.546231 [INFO] (child) spawning: /bin/systemctl reload nginx","@version":"1","@timestamp":"2023-07-07T14:48:04.000Z","syslog_timestamp":"Jul  7 14:48:04","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51209118},"syslog_severity":"notice","received_at":"2023-07-07T14:48:05.072Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 105
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":' in line and '[INFO] (runner) rendered' in line and '"/var/lib/consul-template/publishing-nginx.http.conf.tpl' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:04.644332 [INFO] (runner) rendered \"/var/lib/consul-template/publishing-nginx.http.conf.tpl\" => \"/etc/nginx/conf.d/publishing-nginx.http.conf\"","@version":"1","@timestamp":"2023-07-07T14:48:04.000Z","syslog_timestamp":"Jul  7 14:48:04","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51209768},"syslog_severity":"notice","received_at":"2023-07-07T14:48:05.072Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 106
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":' in line and '[INFO] (runner) rendered' in line and '/var/lib/consul-template/web-nginx.http.conf.tpl' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-69 >> "{"message":"[2023-07-07T13:25:15,231][DEBUG][org.apache.http.wire     ] http-outgoing-11 >> \"{\"syslog_timestamp\":\"Jul  7 13:25:13\",\"message\":\"2023/07/07 13:25:13.759994 [INFO] (runner) rendered \\\"/var/lib/consul-template/web-nginx.http.conf.tpl\\\" => \\\"/etc/nginx/conf.d/web-nginx.http.conf\\\"\",\"syslog_hostname\":\"ip-10-30-140-223\",\"syslog_program\":\"consul-template\",\"@version\":\"1\",\"received_at\":\"2023-07-07T13:25:14.155Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-140-223\",\"ephemeral_id\":\"a75fd1a3-e5d6-4e0b-9d40-a6da5420185d\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-140-223\",\"type\":\"filebeat\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-140-223\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"1171523\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:25:13.000Z\",\"log\":{\"offset\":65548233,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:05.249Z","host":{"name":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"service":{"type":"logstash"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":89917046},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 107
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":' in line and '[INFO] (runner) rendered' in line and '"/var/lib/consul-template/web-nginx.http.conf.tpl' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:04.543147 [INFO] (runner) rendered \"/var/lib/consul-template/web-nginx.http.conf.tpl\" => \"/etc/nginx/conf.d/web-nginx.http.conf\"","@version":"1","@timestamp":"2023-07-07T14:48:04.000Z","syslog_timestamp":"Jul  7 14:48:04","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":30393527},"syslog_severity":"notice","received_at":"2023-07-07T14:48:04.703Z","agent":{"hostname":"ip-10-30-140-225","type":"filebeat","name":"ip-10-30-140-225","version":"7.11.2","ephemeral_id":"4149845f-c923-4187-b528-2d5a729e6ae3","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-225\"}","syslog_pid":"3336928","syslog_hostname":"ip-10-30-140-225","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 108
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":' in line and '[INFO] (runner) executing command' in line and '/var/lib/consul-template/web-nginx.http.conf.tpl' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:48:04.543664 [INFO] (runner) executing command \"/bin/systemctl reload nginx\" from \"/var/lib/consul-template/web-nginx.http.conf.tpl\" => \"/etc/nginx/conf.d/web-nginx.http.conf\"","@version":"1","@timestamp":"2023-07-07T14:48:04.000Z","syslog_timestamp":"Jul  7 14:48:04","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":30394118},"syslog_severity":"notice","received_at":"2023-07-07T14:48:04.703Z","agent":{"hostname":"ip-10-30-140-225","type":"filebeat","name":"ip-10-30-140-225","version":"7.11.2","ephemeral_id":"4149845f-c923-4187-b528-2d5a729e6ae3","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-225\"}","syslog_pid":"3336928","syslog_hostname":"ip-10-30-140-225","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 109
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":' in line and '[WARN]  agent: Check is now critical: check=_nomad-check-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:48:04.651Z [WARN]  agent: Check is now critical: check=_nomad-check-151e0f20638b39d046a4fa408a63ddf4ad8d2405","@version":"1","@timestamp":"2023-07-07T14:48:04.000Z","syslog_timestamp":"Jul  7 14:48:04","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":30329493},"syslog_severity":"notice","received_at":"2023-07-07T14:48:04.780Z","agent":{"hostname":"ip-10-30-140-12","type":"filebeat","name":"ip-10-30-140-12","version":"7.11.2","ephemeral_id":"b5815037-0252-4b49-831a-4cc67f6d4c8d","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-12\"}","syslog_pid":"31572","syslog_hostname":"ip-10-30-140-12","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 110
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' << "{"took":' in line and ',"errors":false,"items":[{"index":{"_index":' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-6 << "{"took":6,"errors":false,"items":[{"index":{"_index":"syslog-2023-07-07","_type":"_doc","_id":"3048094012","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":2324136,"_primary_term":1,"status":201}}]}"
        return 111
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":' in line and '[INFO] (runner) executing command' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-69 >> "{"message":"[2023-07-07T13:25:15,232][DEBUG][org.apache.http.wire     ] http-outgoing-11 >> \"{\"syslog_timestamp\":\"Jul  7 13:25:13\",\"message\":\"2023/07/07 13:25:13.885380 [INFO] (runner) executing command \\\"/bin/systemctl reload nginx\\\" from \\\"/var/lib/consul-template/web-nginx.http.conf.tpl\\\" => \\\"/etc/nginx/conf.d/web-nginx.http.conf\\\"\",\"syslog_hostname\":\"ip-10-30-140-223\",\"syslog_program\":\"consul-template\",\"@version\":\"1\",\"received_at\":\"2023-07-07T13:25:14.156Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-140-223\",\"ephemeral_id\":\"a75fd1a3-e5d6-4e0b-9d40-a6da5420185d\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-140-223\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-140-223\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"1171523\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:25:13.000Z\",\"log\":{\"offset\":65549876,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:06.869Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":89928556},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 112
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":' in line and '[INFO] (child) spawning: /bin/systemctl reload nginx' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-70 >> "{"message":"[2023-07-07T13:25:15,232][DEBUG][org.apache.http.wire     ] http-outgoing-11 >> \"{\"syslog_timestamp\":\"Jul  7 13:25:13\",\"message\":\"2023/07/07 13:25:13.885752 [INFO] (child) spawning: /bin/systemctl reload nginx\",\"syslog_hostname\":\"ip-10-30-140-223\",\"syslog_program\":\"consul-template\",\"@version\":\"1\",\"received_at\":\"2023-07-07T13:25:14.156Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-140-223\",\"ephemeral_id\":\"a75fd1a3-e5d6-4e0b-9d40-a6da5420185d\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-140-223\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-140-223\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"1171523\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:25:13.000Z\",\"log\":{\"offset\":65550340,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:07.215Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":89930999},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"module":"logstash","timezone":"+00:00","dataset":"logstash.log"},"fileset":{"name":"log"}}[\n]"
        return 113
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"syslog_severity_code":5,"message":"INFO [monitoring] log/log.go:144 Non-zero metrics in the last 30s' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"INFO [monitoring] log/log.go:144 Non-zero metrics in the last 30s {\"monitoring\": {\"metrics\": {\"beat\":{\"cgroup\":{\"cpuacct\":{\"total\":{\"ns\":15668164}},\"memory\":{\"mem\":{\"usage\":{\"bytes\":4096}}}},\"cpu\":{\"system\":{\"ticks\":5673370,\"time\":{\"ms\":6}},\"total\":{\"ticks\":15040260,\"time\":{\"ms\":17},\"value\":15040260},\"user\":{\"ticks\":9366890,\"time\":{\"ms\":11}}},\"handles\":{\"limit\":{\"hard\":524288,\"soft\":1024},\"open\":18},\"info\":{\"ephemeral_id\":\"941f35eb-1b63-419f-9b53-f13715239ce1\",\"uptime\":{\"ms\":23079810927}},\"memstats\":{\"gc_next\":22979520,\"memory_alloc\":18004080,\"memory_total\":3887530428216,\"rss\":52211712},\"runtime\":{\"goroutines\":56}},\"filebeat\":{\"events\":{\"added\":7,\"done\":7},\"harvester\":{\"open_files\":1,\"running\":1}},\"libbeat\":{\"config\":{\"module\":{\"running\":3}},\"output\":{\"events\":{\"acked\":7,\"active\":0,\"batches\":2,\"total\":7},\"read\":{\"bytes\":12},\"write\":{\"bytes\":2047}},\"pipeline\":{\"clients\":3,\"events\":{\"active\":0,\"published\":7,\"total\":7},\"queue\":{\"acked\":7}}},\"registrar\":{\"states\":{\"current\":3,\"update\":7},\"writes\":{\"success\":2,\"total\":2}},\"system\":{\"load\":{\"1\":0.08,\"15\":0.08,\"5\":0.05,\"norm\":{\"1\":0.04,\"15\":0.04,\"5\":0.025}}}}}}","@version":"1","@timestamp":"2023-07-07T14:48:05.000Z","syslog_timestamp":"Jul  7 14:48:05","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":4292482},"syslog_severity":"notice","received_at":"2023-07-07T14:48:06.664Z","agent":{"hostname":"ip-10-30-142-40","type":"filebeat","name":"ip-10-30-142-40","version":"7.11.2","ephemeral_id":"941f35eb-1b63-419f-9b53-f13715239ce1","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-142-40\"}","syslog_pid":"509","syslog_hostname":"ip-10-30-142-40","syslog_facility":"user-level","syslog_program":"filebeat"}[\n]"
        return 114
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'error from graph DB' in line and '"dp-hierarchy-builder' in line and 'write: broken pipe' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-70 >> "{"message":"[2023-07-07T13:42:29,936][DEBUG][org.apache.http.wire     ] http-outgoing-6 >> \"{\"event\":\"error from graph DB\",\"namespace\":\"dp-hierarchy-builder\",\"@version\":\"1\",\"created_at\":\"2023-07-07T13:22:34.302273101Z\",\"severity\":1,\"errors\":[{\"message\":\"write tcp 172.17.0.10:39146->10.30.136.184:8182: write: broken pipe\",\"stack_trace\":[{\"file\":\"/go/pkg/mod/github.com/!o!n!sdigital/log.go/v2@v2.4.1/log/log.go\",\"line\":93,\"function\":\"github.com/ONSdigital/log.go/v2/log.Error\"},{\"file\":\"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.14.0/graph/error_consumer.go\",\"line\":20,\"function\":\"github.com/ONSdigital/dp-graph/v2/graph.NewLoggingErrorConsumer.func1\"},{\"file\":\"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.14.0/graph/error_consumer.go\",\"line\":38,\"function\":\"github.com/ONSdigital/dp-graph/v2/graph.NewErrorConsumer.func1\"},{\"file\":\"/usr/local/go/src/runtime/asm_amd64.s\",\"line\":1598,\"function\":\"runtime.goexit\"}]}],\"tags\":[],\"@timestamp\":\"2023-07-07T13:22:34.302Z\",\"nomad\":{\"allocation_id\":\"047b7bda-e889-43c6-be24-cf50ca9db2c6\",\"host\":\"ip-10-30-138-245\"}}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:07.882Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":29473824},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 115
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[WARN] (view) health.service(dp-charts-api|passing):' in line and 'remote error: tls: bad certificate (retry attempt' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-69 >> "{"message":"[2023-07-07T13:30:31,294][DEBUG][org.apache.http.wire     ] http-outgoing-18 >> \"{\"syslog_timestamp\":\"Jul  7 13:30:30\",\"message\":\"2023/07/07 13:30:30.217294 [WARN] (view) health.service(dp-charts-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-charts-api?index=40675393&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 877753 after \\\"20s\\\")\",\"syslog_hostname\":\"ip-10-30-138-60\",\"syslog_program\":\"consul-template\",\"@version\":\"1\",\"received_at\":\"2023-07-07T13:30:30.259Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-138-60\",\"ephemeral_id\":\"933a59a1-6de1-49e1-ac6c-dc3b6ffd15d4\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-138-60\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-138-60\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"13473\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:30:30.000Z\",\"log\":{\"offset\":41559888,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:10.311Z","host":{"name":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"service":{"type":"logstash"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":73117701},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 116
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"(view) health.service(dp-charts-api|passing):' in line and 'remote error: tls: bad certificate (retry attempt' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-70 >> "{"message":"[2023-07-07T13:30:31,294][DEBUG][org.apache.http.wire     ] http-outgoing-18 >> \"{\"syslog_timestamp\":\"Jul  7 13:30:30\",\"message\":\"(view) health.service(dp-charts-api|passing): Get https://127.0.0.1:8500/v1/health/service/dp-charts-api?index=40675393&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 877753 after \\\"20s\\\")\",\"syslog_hostname\":\"ip-10-30-138-60\",\"syslog_program\":\"consul-template\",\"@version\":\"1\",\"received_at\":\"2023-07-07T13:30:30.259Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-138-60\",\"ephemeral_id\":\"933a59a1-6de1-49e1-ac6c-dc3b6ffd15d4\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-138-60\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-138-60\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"13473\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:30:30.000Z\",\"log\":{\"offset\":41560199,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:10.333Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":73118995},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 117
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"nginxlog-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-70 >> "{"message":"[2023-07-07T14:24:52,049][DEBUG][org.apache.http.wire     ] http-outgoing-12 << \"{\"took\":5,\"errors\":false,\"items\":[{\"index\":{\"_index\":\"nginxlog-2023-07-07\",\"_type\":\"_doc\",\"_id\":\"2289217152\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":2,\"successful\":2,\"failed\":0},\"_seq_no\":1465278,\"_primary_term\":1,\"status\":201}}]}\"","@version":"1","@timestamp":"2023-07-07T14:48:10.469Z","host":{"name":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"service":{"type":"logstash"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":11223246},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"module":"logstash","dataset":"logstash.log","timezone":"+00:00"},"fileset":{"name":"log"}}[\n]"
        return 118
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"INFO [publisher] pipeline/retry.go:223   done' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-70 >> "{"message":"[2023-07-07T13:37:22,535][DEBUG][org.apache.http.wire     ] http-outgoing-31 >> \"{\"message\":\"[2023-07-07T13:14:47,358][DEBUG][org.apache.http.wire     ] http-outgoing-23 >> \\\"{\\\"message\\\":\\\"[2023-07-07T13:11:09,789][DEBUG][org.apache.http.wire     ] http-outgoing-3 >> \\\\\\\"{\\\\\\\"syslog_timestamp\\\\\\\":\\\\\\\"Jul  7 13:09:45\\\\\\\",\\\\\\\"message\\\\\\\":\\\\\\\"INFO [publisher] pipeline/retry.go:223   done\\\\\\\",\\\\\\\"syslog_hostname\\\\\\\":\\\\\\\"ip-10-30-140-69\\\\\\\",\\\\\\\"syslog_program\\\\\\\":\\\\\\\"filebeat\\\\\\\",\\\\\\\"@version\\\\\\\":\\\\\\\"1\\\\\\\",\\\\\\\"received_at\\\\\\\":\\\\\\\"2023-07-07T13:09:45.901Z\\\\\\\",\\\\\\\"ecs\\\\\\\":{\\\\\\\"version\\\\\\\":\\\\\\\"1.6.0\\\\\\\"},\\\\\\\"agent\\\\\\\":{\\\\\\\"name\\\\\\\":\\\\\\\"ip-10-30-140-69\\\\\\\",\\\\\\\"ephemeral_id\\\\\\\":\\\\\\\"7a6d010c-bb9f-4225-b6de-c6b2909d45c4\\\\\\\",\\\\\\\"id\\\\\\\":\\\\\\\"a2790c7c-7e0d-4074-a309-2c69835809e9\\\\\\\",\\\\\\\"type\\\\\\\":\\\\\\\"filebeat\\\\\\\",\\\\\\\"version\\\\\\\":\\\\\\\"7.11.2\\\\\\\",\\\\\\\"hostname\\\\\\\":\\\\\\\"ip-10-30-140-69\\\\\\\"},\\\\\\\"received_from\\\\\\\":\\\\\\\"{\\\\\\\\\\\\\\\"name\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"ip-10-30-140-69\\\\\\\\\\\\\\\"}\\\\\\\",\\\\\\\"fields\\\\\\\":{\\\\\\\"type\\\\\\\":\\\\\\\"system\\\\\\\"},\\\\\\\"syslog_pid\\\\\\\":\\\\\\\"8983\\\\\\\",\\\\\\\"syslog_severity\\\\\\\":\\\\\\\"notice\\\\\\\",\\\\\\\"syslog_facility\\\\\\\":\\\\\\\"user-level\\\\\\\",\\\\\\\"tags\\\\\\\":[\\\\\\\"beats_input_codec_plain_applied\\\\\\\"],\\\\\\\"syslog_facility_code\\\\\\\":1,\\\\\\\"input\\\\\\\":{\\\\\\\"type\\\\\\\":\\\\\\\"log\\\\\\\"},\\\\\\\"@timestamp\\\\\\\":\\\\\\\"2023-07-07T13:09:45.000Z\\\\\\\",\\\\\\\"log\\\\\\\":{\\\\\\\"offset\\\\\\\":70595133,\\\\\\\"file\\\\\\\":{\\\\\\\"path\\\\\\\":\\\\\\\"/var/log/syslog\\\\\\\"}},\\\\\\\"syslog_severity_code\\\\\\\":5}[\\\\\\\\n]\\\\\\\"\\\",\\\"event\\\":{\\\"dataset\\\":\\\"logstash.log\\\",\\\"timezone\\\":\\\"+00:00\\\",\\\"module\\\":\\\"logstash\\\"},\\\"@version\\\":\\\"1\\\",\\\"ecs\\\":{\\\"version\\\":\\\"1.7.0\\\"},\\\"agent\\\":{\\\"name\\\":\\\"ip-10-30-142-35\\\",\\\"ephemeral_id\\\":\\\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\\\",\\\"id\\\":\\\"c81ff488-13bd-4915-83df-f9dd60fa5444\\\",\\\"hostname\\\":\\\"ip-10-30-142-35\\\",\\\"type\\\":\\\"filebeat\\\",\\\"version\\\":\\\"7.11.2\\\"},\\\"fileset\\\":{\\\"name\\\":\\\"log\\\"},\\\"tags\\\":[\\\"beats_input_codec_plain_applied\\\"],\\\"input\\\":{\\\"type\\\":\\\"log\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:14:47.049Z\\\",\\\"host\\\":{\\\"name\\\":\\\"ip-10-30-142-35\\\"},\\\"log\\\":{\\\"offset\\\":63356270,\\\"file\\\":{\\\"path\\\":\\\"/var/log/logstash/logstash-plain.log\\\"}},\\\"service\\\":{\\\"type\\\":\\\"logstash\\\"}}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:37:22.377Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":91079013,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:13.842Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":59071849},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 119
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"WARNING:Unknown index ' in line and 'seen, reloading interface list' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"WARNING:Unknown index 261582 seen, reloading interface list","@version":"1","@timestamp":"2023-07-07T14:48:15.000Z","syslog_timestamp":"Jul  7 14:48:15","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":79463091},"syslog_severity":"notice","received_at":"2023-07-07T14:48:16.469Z","agent":{"hostname":"ip-10-30-140-69","type":"filebeat","name":"ip-10-30-140-69","version":"7.11.2","ephemeral_id":"7a6d010c-bb9f-4225-b6de-c6b2909d45c4","id":"a2790c7c-7e0d-4074-a309-2c69835809e9"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-69\"}","syslog_pid":"634","syslog_hostname":"ip-10-30-140-69","syslog_facility":"user-level","syslog_program":"networkd-dispatcher"}[\n]"
        return 120
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"ERROR:Unknown interface index' in line and 'seen even after reload' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"ERROR:Unknown interface index 261582 seen even after reload","@version":"1","@timestamp":"2023-07-07T14:48:15.000Z","syslog_timestamp":"Jul  7 14:48:15","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":79463973},"syslog_severity":"notice","received_at":"2023-07-07T14:48:16.469Z","agent":{"hostname":"ip-10-30-140-69","type":"filebeat","name":"ip-10-30-140-69","version":"7.11.2","ephemeral_id":"7a6d010c-bb9f-4225-b6de-c6b2909d45c4","id":"a2790c7c-7e0d-4074-a309-2c69835809e9"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-69\"}","syslog_pid":"634","syslog_hostname":"ip-10-30-140-69","syslog_facility":"user-level","syslog_program":"networkd-dispatcher"}[\n]"
        return 121
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"failed to update services in Consul' in line and '"Unexpected response code: 500 (Unknown check' in line and '"_nomad-check-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"    {\"@level\":\"warn\",\"@message\":\"failed to update services in Consul\",\"@module\":\"consul.sync\",\"@timestamp\":\"2023-07-07T14:48:15.841615Z\",\"error\":\"Unexpected response code: 500 (Unknown check \\\"_nomad-check-f111e4be844446b7f9b5c4d6aa3f52b3e89a66a4\\\")\"}","@version":"1","@timestamp":"2023-07-07T14:48:15.000Z","syslog_timestamp":"Jul  7 14:48:15","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":79465102},"syslog_severity":"notice","received_at":"2023-07-07T14:48:16.469Z","agent":{"hostname":"ip-10-30-140-69","type":"filebeat","name":"ip-10-30-140-69","version":"7.11.2","ephemeral_id":"7a6d010c-bb9f-4225-b6de-c6b2909d45c4","id":"a2790c7c-7e0d-4074-a309-2c69835809e9"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-69\"}","syslog_pid":"1091453","syslog_hostname":"ip-10-30-140-69","syslog_facility":"user-level","syslog_program":"nomad"}[\n]"
        return 122
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  agent: Deregistered service: service=_nomad-task-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:48:15.840Z [INFO]  agent: Deregistered service: service=_nomad-task-61b8e259-e62a-e785-a95c-e50f1e382a82-dp-nlp-category-api-web-dp-nlp-category-api-http","@version":"1","@timestamp":"2023-07-07T14:48:15.000Z","syslog_timestamp":"Jul  7 14:48:15","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":79465910},"syslog_severity":"notice","received_at":"2023-07-07T14:48:16.469Z","agent":{"hostname":"ip-10-30-140-69","type":"filebeat","name":"ip-10-30-140-69","version":"7.11.2","ephemeral_id":"7a6d010c-bb9f-4225-b6de-c6b2909d45c4","id":"a2790c7c-7e0d-4074-a309-2c69835809e9"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-69\"}","syslog_pid":"951690","syslog_hostname":"ip-10-30-140-69","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 123
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[ERROR] agent.http: Request error: method=PUT url=/v1/agent/check/deregister/_nomad-check' in line and '"Unknown check ' in line and '"_nomad-check-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:48:15.841Z [ERROR] agent.http: Request error: method=PUT url=/v1/agent/check/deregister/_nomad-check-f111e4be844446b7f9b5c4d6aa3f52b3e89a66a4 from=127.0.0.1:53134 error=\"Unknown check \"_nomad-check-f111e4be844446b7f9b5c4d6aa3f52b3e89a66a4\"\"","@version":"1","@timestamp":"2023-07-07T14:48:15.000Z","syslog_timestamp":"Jul  7 14:48:15","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":79466130},"syslog_severity":"notice","received_at":"2023-07-07T14:48:16.470Z","agent":{"hostname":"ip-10-30-140-69","type":"filebeat","name":"ip-10-30-140-69","version":"7.11.2","ephemeral_id":"7a6d010c-bb9f-4225-b6de-c6b2909d45c4","id":"a2790c7c-7e0d-4074-a309-2c69835809e9"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-69\"}","syslog_pid":"951690","syslog_hostname":"ip-10-30-140-69","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 124
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"event":"error from graph DB","error":{"message":"write tcp ' in line and 'write: broken pipe"' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-48 >> "{"@version":"1","@timestamp":"2023-07-07T14:48:14.021Z","created_at":"2023-07-07T14:48:14.021104478Z","namespace":"./dp-hierarchy-api","nomad":{"allocation_id":"37fa8103-0ef6-262c-638f-bd5fc1d390a3","host":"ip-10-30-140-223"},"tags":[],"severity":1,"event":"error from graph DB","error":{"message":"write tcp 172.17.0.3:39584->10.30.137.8:8182: write: broken pipe","data":"{\n\n}","stack_trace":[{"function":"github.com/ONSdigital/dp-graph/v2/graph.NewLoggingErrorConsumer.func1","file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.12.0/graph/error_consumer.go","line":20},{"function":"github.com/ONSdigital/dp-graph/v2/graph.NewErrorConsumer.func1","file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.12.0/graph/error_consumer.go","line":38},{"function":"runtime.goexit","file":"/usr/local/go/src/runtime/asm_amd64.s","line":1594}]}}[\n]"
        return 125
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  agent: Synced check: check=_nomad-check-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:48:15.905Z [INFO]  agent: Synced check: check=_nomad-check-68f3176f0cd427d16a5585de853e23f68db4e6a4","@version":"1","@timestamp":"2023-07-07T14:48:15.000Z","syslog_timestamp":"Jul  7 14:48:15","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":72557128},"syslog_severity":"notice","received_at":"2023-07-07T14:48:16.390Z","agent":{"hostname":"ip-10-30-140-223","type":"filebeat","name":"ip-10-30-140-223","version":"7.11.2","ephemeral_id":"a75fd1a3-e5d6-4e0b-9d40-a6da5420185d","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-140-223\"}","syslog_pid":"1188980","syslog_hostname":"ip-10-30-140-223","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 126
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'Failed to update stats for container' in line and 'unable to determine device info for dir: /var/lib/docker/' in line and 'diff: stat failed on /var/lib/docker/' in line and '/diff with error: no such file or directory, continuing to push stats' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"W0707 14:48:16.327466   12041 container.go:549] Failed to update stats for container \"/docker/e9484b6118c44cfb7ff1e3a3fb9c10919336dac3a69dd39aee313b817e730098\": unable to determine device info for dir: /var/lib/docker/165536.165536/overlay2/1dc08eb07ad6a9c792de03bbe76a481a27cd156f6d7dfc839ed3f66c20b56e92/diff: stat failed on /var/lib/docker/165536.165536/overlay2/1dc08eb07ad6a9c792de03bbe76a481a27cd156f6d7dfc839ed3f66c20b56e92/diff with error: no such file or directory, continuing to push stats","@version":"1","@timestamp":"2023-07-07T14:48:16.000Z","syslog_timestamp":"Jul  7 14:48:16","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":72829735},"syslog_severity":"notice","received_at":"2023-07-07T14:48:16.800Z","agent":{"hostname":"ip-10-30-138-203","type":"filebeat","name":"ip-10-30-138-203","version":"7.11.2","ephemeral_id":"839041a5-29e2-4e7a-8063-efa3f5ee509e","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-203\"}","syslog_pid":"12041","syslog_hostname":"ip-10-30-138-203","syslog_facility":"user-level","syslog_program":"cadvisor"}[\n]"
        return 127
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"message":"INFO log/harvester.go:' in line and ' Harvester started for file: /var/log/logstash/logstash-plain.log"' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"INFO log/harvester.go:302 Harvester started for file: /var/log/logstash/logstash-plain.log","@version":"1","@timestamp":"2023-07-07T14:48:12.000Z","syslog_timestamp":"Jul  7 14:48:12","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":2204809},"syslog_severity":"notice","received_at":"2023-07-07T14:48:18.002Z","agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-142-35\"}","syslog_pid":"483","syslog_hostname":"ip-10-30-142-35","syslog_facility":"user-level","syslog_program":"filebeat"}[\n]"
        return 128
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"message":"INFO [monitoring] log/log.go:184 Non-zero metrics in the last 30s' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-6 >> "{"syslog_severity_code":5,"message":"INFO [monitoring] log/log.go:184 Non-zero metrics in the last 30s {\"monitoring\": {\"metrics\": {\"beat\":{\"cgroup\":{\"cpuacct\":{\"total\":{\"ns\":76498056}},\"memory\":{\"mem\":{\"usage\":{\"bytes\":-45056}}}},\"cpu\":{\"system\":{\"ticks\":6032500},\"total\":{\"ticks\":31941240,\"time\":{\"ms\":76},\"value\":31941240},\"user\":{\"ticks\":25908740,\"time\":{\"ms\":76}}},\"handles\":{\"limit\":{\"hard\":524288,\"soft\":1024},\"open\":21},\"info\":{\"ephemeral_id\":\"3904140f-ab6d-4ad6-88f7-dace9a6792fe\",\"uptime\":{\"ms\":12892470257},\"version\":\"7.17.7\"},\"memstats\":{\"gc_next\":21847816,\"memory_alloc\":14267264,\"memory_total\":8553842179312,\"rss\":110501888},\"runtime\":{\"goroutines\":96}},\"filebeat\":{\"events\":{\"active\":1,\"added\":55,\"done\":54},\"harvester\":{\"open_files\":4,\"running\":4}},\"libbeat\":{\"config\":{\"module\":{\"running\":7}},\"output\":{\"events\":{\"acked\":54,\"active\":1,\"batches\":15,\"total\":55},\"read\":{\"bytes\":84},\"write\":{\"bytes\":10974}},\"pipeline\":{\"clients\":7,\"events\":{\"active\":5,\"published\":55,\"total\":55},\"queue\":{\"acked\":54}}},\"registrar\":{\"states\":{\"current\":6,\"update\":54},\"writes\":{\"success\":31,\"total\":31}},\"system\":{\"load\":{\"1\":0,\"15\":0,\"5\":0,\"norm\":{\"1\":0,\"15\":0,\"5\":0}}}}}}","@version":"1","@timestamp":"2023-07-07T14:48:17.000Z","syslog_timestamp":"Jul  7 14:48:17","ecs":{"version":"1.12.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":2208091},"syslog_severity":"notice","received_at":"2023-07-07T14:48:18.423Z","agent":{"hostname":"ip-10-30-137-38","type":"filebeat","name":"ip-10-30-137-38","version":"7.17.7","ephemeral_id":"3904140f-ab6d-4ad6-88f7-dace9a6792fe","id":"a1437f6b-3a75-4735-8e10-b8d93f5fbc9d"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-137-38\"}","syslog_pid":"5007","syslog_hostname":"ip-10-30-137-38","syslog_facility":"user-level","syslog_program":"filebeat"}[\n]"
        return 129
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"errors":[{"message":"write tcp ' in line and 'write: broken pipe' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-61 >> "{"@version":"1","@timestamp":"2023-07-07T14:48:17.691Z","errors":[{"message":"write tcp 172.17.0.21:35534->10.30.136.184:8182: write: broken pipe","stack_trace":[{"function":"github.com/ONSdigital/log.go/v2/log.Error","file":"/go/pkg/mod/github.com/!o!n!sdigital/log.go/v2@v2.4.1/log/log.go","line":93},{"function":"github.com/ONSdigital/dp-graph/v2/graph.NewLoggingErrorConsumer.func1","file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.15.0/graph/error_consumer.go","line":20},{"function":"github.com/ONSdigital/dp-graph/v2/graph.NewErrorConsumer.func1","file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.15.0/graph/error_consumer.go","line":38},{"function":"runtime.goexit","file":"/usr/local/go/src/runtime/asm_amd64.s","line":1598}]}],"created_at":"2023-07-07T14:48:17.69183202Z","namespace":"dp-observation-importer","nomad":{"allocation_id":"f5f3304c-7673-fc27-75c7-850de3bce688","host":"ip-10-30-138-234"},"tags":[],"severity":1,"event":"error from graph DB"}[\n]"
        return 130
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'INFO log/harvester.go:' in line and ' File is inactive: /var/' in line and 'Closing because close_inactive of 5m0s reached.' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-77 >> "{"message":"[2023-07-07T14:11:41,610][DEBUG][org.apache.http.wire     ] http-outgoing-3 >> \"{\"syslog_timestamp\":\"Jul  7 14:11:41\",\"message\":\"INFO log/harvester.go:333 File is inactive: /var/log/audit/audit.log. Closing because close_inactive of 5m0s reached.\",\"syslog_hostname\":\"ip-10-30-140-69\",\"syslog_program\":\"filebeat\",\"@version\":\"1\",\"received_at\":\"2023-07-07T14:11:41.362Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-140-69\",\"ephemeral_id\":\"7a6d010c-bb9f-4225-b6de-c6b2909d45c4\",\"id\":\"a2790c7c-7e0d-4074-a309-2c69835809e9\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-140-69\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-140-69\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"8983\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:11:41.000Z\",\"log\":{\"offset\":76207032,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:22.472Z","host":{"name":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"service":{"type":"logstash"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":22140080},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 131
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"error from graph DB' in line and '"write tcp ' in line and 'write: broken pipe' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-78 >> "{"message":"[2023-07-07T13:52:08,928][DEBUG][org.apache.http.wire     ] http-outgoing-8 >> \"{\"event\":\"error from graph DB\",\"namespace\":\"./dp-hierarchy-api\",\"@version\":\"1\",\"created_at\":\"2023-07-07T13:52:02.36766629Z\",\"severity\":1,\"error\":{\"message\":\"write tcp 172.17.0.40:46466->10.30.136.184:8182: write: broken pipe\",\"stack_trace\":[{\"file\":\"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.12.0/graph/error_consumer.go\",\"line\":20,\"function\":\"github.com/ONSdigital/dp-graph/v2/graph.NewLoggingErrorConsumer.func1\"},{\"file\":\"/go/pkg/mod/github.com/!o!n!sdigital/dp-graph/v2@v2.12.0/graph/error_consumer.go\",\"line\":38,\"function\":\"github.com/ONSdigital/dp-graph/v2/graph.NewErrorConsumer.func1\"},{\"file\":\"/usr/local/go/src/runtime/asm_amd64.s\",\"line\":1594,\"function\":\"runtime.goexit\"}],\"data\":\"{\\n\\n}\"},\"tags\":[],\"@timestamp\":\"2023-07-07T13:52:02.367Z\",\"nomad\":{\"allocation_id\":\"a825c934-fb19-e9a0-c9de-7788305723a9\",\"host\":\"ip-10-30-141-96\"}}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:25.767Z","host":{"name":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"service":{"type":"logstash"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":36990807},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 132
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-71 >> "{"message":"[2023-07-07T14:38:20,474][DEBUG][org.apache.http.wire     ] http-outgoing-18 << \"{\"took\":11,\"errors\":false,\"items\":[{\"index\":{\"_index\":\"syslog-2023-07-07\",\"_type\":\"_doc\",\"_id\":\"2721419550\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":2,\"successful\":2,\"failed\":0},\"_seq_no\":2297574,\"_primary_term\":1,\"status\":201}}]}\"","@version":"1","@timestamp":"2023-07-07T14:48:28.235Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":5224347},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"timezone":"+00:00","dataset":"logstash.log","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 133
    # if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and ' >> "{"message":"    raise HTTP_EXCEPTIONS.get(status_code, TransportError)(status_code, error_message, additional_info)"' in line and '"nomad":{"allocation_id":"' in line:
    #     # this copes with all variants of:
    #     # [DEBUG][org.apache.http.wire     ] http-outgoing-61 >> "{"message":"    raise HTTP_EXCEPTIONS.get(status_code, TransportError)(status_code, error_message, additional_info)","@version":"1","@timestamp":"2023-07-07T14:48:29.613Z","nomad":{"allocation_id":"c5a93b20-2f60-d654-013e-a55b2fb7e644","host":"ip-10-30-138-203"},"fields":{"type":"application"},"tags":[]}[\n]"
    #     return 134
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"GraphQL error: Must provide an operation.' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-61 >> "{"@version":"1","@timestamp":"2023-07-07T14:48:29.116Z","namespace":"metadata","inst":"ip-10-30-137-38","level":"info","tags":[],"msg":"GraphQL error: Must provide an operation.","event":"message"}[\n]"
        return 135
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[WARN]  agent: Check is now critical: check=_nomad-check-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-78 >> "{"message":"[2023-07-07T14:22:52,090][DEBUG][org.apache.http.wire     ] http-outgoing-55 >> \"{\"message\":\"[2023-07-07T13:24:15,858][DEBUG][org.apache.http.wire     ] http-outgoing-3 >> \\\"{\\\"syslog_timestamp\\\":\\\"Jul  7 13:24:14\\\",\\\"message\\\":\\\"    2023-07-07T13:24:14.903Z [WARN]  agent: Check is now critical: check=_nomad-check-433f6f1b446475e25558ceaf5b524ad4508bb203\\\",\\\"syslog_hostname\\\":\\\"ip-10-30-138-203\\\",\\\"syslog_program\\\":\\\"consul\\\",\\\"@version\\\":\\\"1\\\",\\\"received_at\\\":\\\"2023-07-07T13:24:15.269Z\\\",\\\"ecs\\\":{\\\"version\\\":\\\"1.6.0\\\"},\\\"agent\\\":{\\\"name\\\":\\\"ip-10-30-138-203\\\",\\\"ephemeral_id\\\":\\\"839041a5-29e2-4e7a-8063-efa3f5ee509e\\\",\\\"id\\\":\\\"c81ff488-13bd-4915-83df-f9dd60fa5444\\\",\\\"version\\\":\\\"7.11.2\\\",\\\"hostname\\\":\\\"ip-10-30-138-203\\\",\\\"type\\\":\\\"filebeat\\\"},\\\"received_from\\\":\\\"{\\\\\\\"name\\\\\\\":\\\\\\\"ip-10-30-138-203\\\\\\\"}\\\",\\\"fields\\\":{\\\"type\\\":\\\"system\\\"},\\\"syslog_pid\\\":\\\"2913053\\\",\\\"syslog_severity\\\":\\\"notice\\\",\\\"syslog_facility\\\":\\\"user-level\\\",\\\"tags\\\":[\\\"beats_input_codec_plain_applied\\\"],\\\"syslog_facility_code\\\":1,\\\"input\\\":{\\\"type\\\":\\\"log\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:24:14.000Z\\\",\\\"log\\\":{\\\"offset\\\":65745069,\\\"file\\\":{\\\"path\\\":\\\"/var/log/syslog\\\"}},\\\"syslog_severity_code\\\":5}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:22:51.731Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":78015822,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:48:32.987Z","host":{"name":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"service":{"type":"logstash"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":12722544},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 136
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  agent: Synced service: service=_nomad-task-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-6 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:50:42.766Z [INFO]  agent: Synced service: service=_nomad-task-c5a93b20-2f60-d654-013e-a55b2fb7e644-dp-nlp-category-api-publishing-dp-nlp-category-api-http","@version":"1","@timestamp":"2023-07-07T14:50:42.000Z","syslog_timestamp":"Jul  7 14:50:42","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":73008022},"syslog_severity":"notice","received_at":"2023-07-07T14:50:42.896Z","agent":{"hostname":"ip-10-30-138-203","type":"filebeat","name":"ip-10-30-138-203","version":"7.11.2","ephemeral_id":"839041a5-29e2-4e7a-8063-efa3f5ee509e","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-203\"}","syslog_pid":"2913053","syslog_hostname":"ip-10-30-138-203","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 137
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '(view) health.service' in line and 'remote error: tls: bad certificate (retry attempt' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-6 >> "{"syslog_severity_code":5,"message":"2023/07/07 14:50:43.088990 [WARN] (view) health.service(dp-datawrapper-adapter|passing): Get https://127.0.0.1:8500/v1/health/service/dp-datawrapper-adapter?index=42564487&passing=1&stale=&wait=60000ms: remote error: tls: bad certificate (retry attempt 784073 after \"20s\")","@version":"1","@timestamp":"2023-07-07T14:50:43.000Z","syslog_timestamp":"Jul  7 14:50:43","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":51349241},"syslog_severity":"notice","received_at":"2023-07-07T14:50:43.121Z","agent":{"hostname":"ip-10-30-138-234","type":"filebeat","name":"ip-10-30-138-234","version":"7.11.2","ephemeral_id":"718f058a-ea15-421e-8506-a0fd98a167eb","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-138-234\"}","syslog_pid":"12732","syslog_hostname":"ip-10-30-138-234","syslog_facility":"user-level","syslog_program":"consul-template"}[\n]"
        return 138
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"INFO [monitoring] log/log.go:144 Non-zero metrics in the last 30s' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-78 >> "{"message":"[2023-07-07T14:37:49,226][DEBUG][org.apache.http.wire     ] http-outgoing-3 >> \"{\"syslog_timestamp\":\"Jul  7 14:37:43\",\"message\":\"INFO [monitoring] log/log.go:144 Non-zero metrics in the last 30s {\\\"monitoring\\\": {\\\"metrics\\\": {\\\"beat\\\":{\\\"cgroup\\\":{\\\"cpuacct\\\":{\\\"total\\\":{\\\"ns\\\":6734451}}},\\\"cpu\\\":{\\\"system\\\":{\\\"ticks\\\":11240,\\\"time\\\":{\\\"ms\\\":1}},\\\"total\\\":{\\\"ticks\\\":34490,\\\"time\\\":{\\\"ms\\\":8},\\\"value\\\":34490},\\\"user\\\":{\\\"ticks\\\":23250,\\\"time\\\":{\\\"ms\\\":7}}},\\\"handles\\\":{\\\"limit\\\":{\\\"hard\\\":524288,\\\"soft\\\":1024},\\\"open\\\":20},\\\"info\\\":{\\\"ephemeral_id\\\":\\\"db09980a-8fe7-48b1-8aec-1e4b66ab0d36\\\",\\\"uptime\\\":{\\\"ms\\\":97440668}},\\\"memstats\\\":{\\\"gc_next\\\":20160960,\\\"memory_alloc\\\":15030544,\\\"memory_total\\\":9208357880,\\\"rss\\\":102785024},\\\"runtime\\\":{\\\"goroutines\\\":66}},\\\"filebeat\\\":{\\\"events\\\":{\\\"added\\\":1,\\\"done\\\":1},\\\"harvester\\\":{\\\"open_files\\\":3,\\\"running\\\":3}},\\\"libbeat\\\":{\\\"config\\\":{\\\"module\\\":{\\\"running\\\":3}},\\\"output\\\":{\\\"events\\\":{\\\"acked\\\":1,\\\"active\\\":0,\\\"batches\\\":1,\\\"total\\\":1},\\\"read\\\":{\\\"bytes\\\":6},\\\"write\\\":{\\\"bytes\\\":881}},\\\"pipeline\\\":{\\\"clients\\\":3,\\\"events\\\":{\\\"active\\\":0,\\\"published\\\":1,\\\"total\\\":1},\\\"queue\\\":{\\\"acked\\\":1}}},\\\"registrar\\\":{\\\"states\\\":{\\\"current\\\":3,\\\"update\\\":1},\\\"writes\\\":{\\\"success\\\":1,\\\"total\\\":1}},\\\"system\\\":{\\\"load\\\":{\\\"1\\\":0.53,\\\"15\\\":0.65,\\\"5\\\":0.69,\\\"norm\\\":{\\\"1\\\":0.265,\\\"15\\\":0.325,\\\"5\\\":0.345}}}}}}\",\"syslog_hostname\":\"ip-10-30-142-112\",\"syslog_program\":\"filebeat\",\"@version\":\"1\",\"received_at\":\"2023-07-07T14:37:48.199Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-142-112\",\"ephemeral_id\":\"db09980a-8fe7-48b1-8aec-1e4b66ab0d36\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"hostname\":\"ip-10-30-142-112\",\"type\":\"filebeat\",\"version\":\"7.11.2\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-142-112\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"482\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:37:43.000Z\",\"log\":{\"offset\":2049373,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\"","@version":"1","@timestamp":"2023-07-07T14:50:51.291Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":6545527},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"dataset":"logstash.log","timezone":"+00:00","module":"logstash"},"fileset":{"name":"log"}}[\n]"
        return 139
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/graphql","method":"GET","host":"cantabular-server-web.internal.sandbox","scheme":"http","query":"query={datasets{name}}","port":14020,"status_code":"200"},"aws":{"elb":{"type":"http","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:48:29.582Z","http":{"path":"/graphql","method":"GET","host":"cantabular-server-web.internal.sandbox","scheme":"http","query":"query={datasets{name}}","port":14020,"status_code":"200"},"aws":{"elb":{"type":"http","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a825bd-070909b72ec0f8e16d5074e2","classification":null,"chosen_cert":{"arn":null},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/cant-svr-web-apiext-http-sandbox/cf918bf7791d5672"}}},"elb":{"client":{"ip":"10.30.140.223","user_agent":{"raw":"Go-http-client/1.1"},"port":53326},"duration":{"backend":0.001,"response":0.0,"request":0.0},"bytes_sent":879,"load_balancer":"app/cantabular-server-web-sandbox/bbcd86be1e04853e","backend":{"ip":"10.30.136.254","status_code":"200","port":14020},"http_version":"HTTP/1.1","bytes_received":191,"tls":{"protocol":null,"cipher":null}}}[\n]"
        return 140
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/v10/datasets","method":"GET","host":"cantabular-server-web.internal.sandbox","scheme":"http","query":null,"port":14000,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:49:14.545Z","http":{"path":"/v10/datasets","method":"GET","host":"cantabular-server-web.internal.sandbox","scheme":"http","query":null,"port":14000,"status_code":"200"},"aws":{"elb":{"type":"http","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a825ea-661a615071e93e9d098b7467","classification":null,"chosen_cert":{"arn":null},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/cantabular-svr-web-http-sandbox/ba4e895477ef9f14"}}},"elb":{"client":{"ip":"10.30.140.223","user_agent":{"raw":"Go-http-client/1.1"},"port":44354},"duration":{"backend":0.0,"response":0.0,"request":0.0},"bytes_sent":6369,"load_balancer":"app/cantabular-server-web-sandbox/bbcd86be1e04853e","backend":{"ip":"10.30.136.254","status_code":"200","port":14000},"http_version":"HTTP/1.1","bytes_received":173,"tls":{"protocol":null,"cipher":null}}}[\n]"
        return 141
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"POST' in line and '/graphql' in line and '"cantabular-server-predissemination.internal.sandbox' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-83 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"ecs":{"version":"1.7.0"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"@timestamp":"2023-07-07T14:50:39.349Z","log":{"offset":7484437,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T14:35:20,210][DEBUG][org.apache.http.wire     ] http-outgoing-64 >> \"{\"message\":\"[2023-07-07T13:40:50,457][DEBUG][org.apache.http.wire     ] http-outgoing-8 >> \\\"{\\\"@version\\\":\\\"1\\\",\\\"elb\\\":{\\\"load_balancer\\\":\\\"app/predissemination-sandbox/33e29f9618383559\\\",\\\"duration\\\":{\\\"response\\\":0.0,\\\"backend\\\":0.001,\\\"request\\\":0.0},\\\"tls\\\":{\\\"cipher\\\":null,\\\"protocol\\\":null},\\\"bytes_received\\\":244,\\\"bytes_sent\\\":707,\\\"client\\\":{\\\"port\\\":60262,\\\"user_agent\\\":{\\\"raw\\\":\\\"Go-http-client/1.1\\\"},\\\"ip\\\":\\\"10.30.142.219\\\"},\\\"http_version\\\":\\\"HTTP/1.1\\\",\\\"backend\\\":{\\\"port\\\":14010,\\\"ip\\\":\\\"10.30.142.234\\\",\\\"status_code\\\":\\\"200\\\"}},\\\"http\\\":{\\\"method\\\":\\\"POST\\\",\\\"port\\\":14010,\\\"query\\\":null,\\\"path\\\":\\\"/graphql\\\",\\\"scheme\\\":\\\"http\\\",\\\"host\\\":\\\"cantabular-server-predissemination.internal.sandbox\\\",\\\"status_code\\\":\\\"200\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:36:08.102Z\\\",\\\"aws\\\":{\\\"elb\\\":{\\\"classification_reason\\\":null,\\\"trace_id\\\":\\\"Root=1-64a814c8-57a57a252785b2bf762dde70\\\",\\\"redirect_url\\\":null,\\\"target_group\\\":{\\\"arn\\\":\\\"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/prediss-meta-http-sandbox/1e30f4f144304b2c\\\"},\\\"error\\\":{\\\"reason\\\":null},\\\"chosen_cert\\\":{\\\"arn\\\":null},\\\"type\\\":\\\"http\\\",\\\"classification\\\":null,\\\"action_executed\\\":\\\"forward\\\",\\\"matched_rule_priority\\\":\\\"0\\\"}}}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:35:19.575Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":46571932,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 142
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/florence/login","method":"GET","host":"publishing.dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"},' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-48 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:05.151Z","http":{"path":"/florence/login","method":"GET","host":"publishing.dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"},"aws":{"elb":{"type":"h2","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824f1-59cb1f471b73f8376d5e8787","classification":null,"chosen_cert":{"arn":"arn:aws:acm:eu-west-2:337289154253:certificate/9176a9ed-1235-42da-8ae3-76525c7df2ca"},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/publishing-http-sandbox/20dddaa0ac595874"}}},"elb":{"client":{"ip":"18.169.174.79","user_agent":{"raw":"Blackbox Exporter/0.20.0"},"port":14617},"duration":{"backend":0.002,"response":0.0,"request":0.001},"bytes_sent":1107,"load_balancer":"app/publishing-sandbox/92544f33cb19f1af","backend":{"ip":"10.30.139.85","status_code":"200","port":80},"http_version":"HTTP/2.0","bytes_received":60,"tls":{"protocol":"TLSv1.2","cipher":"ECDHE-RSA-AES128-GCM-SHA256"}}}[\n]"
        return 143
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/","method":"GET","host":"dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:02.003Z","http":{"path":"/","method":"GET","host":"dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"},"aws":{"elb":{"type":"h2","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824ed-4ac9970d080ff28d262a37bb","classification":null,"chosen_cert":{"arn":"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5"},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26"}}},"elb":{"client":{"ip":"3.9.123.111","user_agent":{"raw":"Blackbox Exporter/0.20.0"},"port":7568},"duration":{"backend":0.007,"response":0.0,"request":0.001},"bytes_sent":83132,"load_balancer":"app/web-sandbox/82af646be10141fd","backend":{"ip":"10.30.140.69","status_code":"200","port":80},"http_version":"HTTP/2.0","bytes_received":40,"tls":{"protocol":"TLSv1.2","cipher":"ECDHE-RSA-AES128-GCM-SHA256"}}}[\n]"
        return 144
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/dataset","method":"GET","host":"api.dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:02.446Z","http":{"path":"/dataset","method":"GET","host":"api.dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"},"aws":{"elb":{"type":"h2","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824ee-65dc83164c9e981241e2f056","classification":null,"chosen_cert":{"arn":"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5"},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26"}}},"elb":{"client":{"ip":"18.170.171.177","user_agent":{"raw":"Blackbox Exporter/0.20.0"},"port":25333},"duration":{"backend":0.009,"response":0.0,"request":0.0},"bytes_sent":21070,"load_balancer":"app/web-sandbox/82af646be10141fd","backend":{"ip":"10.30.141.96","status_code":"200","port":80},"http_version":"HTTP/2.0","bytes_received":49,"tls":{"protocol":"TLSv1.2","cipher":"ECDHE-RSA-AES128-GCM-SHA256"}}}[\n]"
        return 145
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/search","method":"GET","host":"api.dp.aws.onsdigital.uk","scheme":"https","query":"q=test","port":443,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:04.436Z","http":{"path":"/search","method":"GET","host":"api.dp.aws.onsdigital.uk","scheme":"https","query":"q=test","port":443,"status_code":"200"},"aws":{"elb":{"type":"h2","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824f0-3d673a9b751445e14b094be5","classification":null,"chosen_cert":{"arn":"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5"},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26"}}},"elb":{"client":{"ip":"18.169.174.79","user_agent":{"raw":"Blackbox Exporter/0.20.0"},"port":63264},"duration":{"backend":0.011,"response":0.0,"request":0.001},"bytes_sent":32037,"load_balancer":"app/web-sandbox/82af646be10141fd","backend":{"ip":"10.30.140.12","status_code":"200","port":80},"http_version":"HTTP/2.0","bytes_received":54,"tls":{"protocol":"TLSv1.2","cipher":"ECDHE-RSA-AES128-GCM-SHA256"}}}[\n]"
        return 146
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/search","method":"GET","host":"dp.aws.onsdigital.uk","scheme":"https","query":"q=test","port":443,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:05.955Z","http":{"path":"/search","method":"GET","host":"dp.aws.onsdigital.uk","scheme":"https","query":"q=test","port":443,"status_code":"200"},"aws":{"elb":{"type":"h2","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824f1-0733fed672c473426ea9e46e","classification":null,"chosen_cert":{"arn":"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5"},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26"}}},"elb":{"client":{"ip":"3.9.123.111","user_agent":{"raw":"Blackbox Exporter/0.20.0"},"port":8782},"duration":{"backend":0.155,"response":0.0,"request":0.001},"bytes_sent":98023,"load_balancer":"app/web-sandbox/82af646be10141fd","backend":{"ip":"10.30.140.69","status_code":"200","port":80},"http_version":"HTTP/2.0","bytes_received":52,"tls":{"protocol":"TLSv1.2","cipher":"ECDHE-RSA-AES128-GCM-SHA256"}}}[\n]"
        return 147
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '","http":{"path":"/","method":"GET","host":"developer.dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:08.833Z","http":{"path":"/","method":"GET","host":"developer.dp.aws.onsdigital.uk","scheme":"https","query":null,"port":443,"status_code":"200"},"aws":{"elb":{"type":"h2","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824f4-7aacae4b24535f120ac57ace","classification":null,"chosen_cert":{"arn":"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5"},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26"}}},"elb":{"client":{"ip":"18.169.174.79","user_agent":{"raw":"Blackbox Exporter/0.20.0"},"port":23139},"duration":{"backend":0.009,"response":0.0,"request":0.001},"bytes_sent":83159,"load_balancer":"app/web-sandbox/82af646be10141fd","backend":{"ip":"10.30.140.225","status_code":"200","port":80},"http_version":"HTTP/2.0","bytes_received":47,"tls":{"protocol":"TLSv1.2","cipher":"ECDHE-RSA-AES128-GCM-SHA256"}}}[\n]"
        return 148
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/v10/datasets","method":"GET","host":"cantabular-server-publishing.internal.sandbox","scheme":"http","query":null,"port":14000,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-49 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:02.967Z","http":{"path":"/v10/datasets","method":"GET","host":"cantabular-server-publishing.internal.sandbox","scheme":"http","query":null,"port":14000,"status_code":"200"},"aws":{"elb":{"type":"http","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824ee-103d8aa131f7ea6a2eae57c5","classification":null,"chosen_cert":{"arn":null},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/cantabular-svr-pub-http-sandbox/283b3081dc9e4cea"}}},"elb":{"client":{"ip":"10.30.138.60","user_agent":{"raw":"Go-http-client/1.1"},"port":50868},"duration":{"backend":0.001,"response":0.0,"request":0.0},"bytes_sent":6369,"load_balancer":"app/cantabular-server-pub-sandbox/b413eb65a1070c92","backend":{"ip":"10.30.137.38","status_code":"200","port":14000},"http_version":"HTTP/1.1","bytes_received":180,"tls":{"protocol":null,"cipher":null}}}[\n]"
        return 149
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/graphql","method":"GET","host":"cantabular-server-publishing.internal.sandbox","scheme":"http","query":"query={datasets{name}}","port":14020,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-49 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:12.083Z","http":{"path":"/graphql","method":"GET","host":"cantabular-server-publishing.internal.sandbox","scheme":"http","query":"query={datasets{name}}","port":14020,"status_code":"200"},"aws":{"elb":{"type":"http","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a824f8-43b42559109ec7b25ea2387d","classification":null,"chosen_cert":{"arn":null},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/cant-pub-apiext-http-sandbox/79d9d659248981a0"}}},"elb":{"client":{"ip":"10.30.139.85","user_agent":{"raw":"Go-http-client/1.1"},"port":38040},"duration":{"backend":0.002,"response":0.0,"request":0.0},"bytes_sent":879,"load_balancer":"app/cantabular-server-pub-sandbox/b413eb65a1070c92","backend":{"ip":"10.30.137.38","status_code":"200","port":14020},"http_version":"HTTP/1.1","bytes_received":198,"tls":{"protocol":null,"cipher":null}}}[\n]"
        return 150
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"http":{"path":"/graphql","method":"GET","host":"cantabular-server-publishing.internal.sandbox","scheme":"http","query":null,"port":14010,"status_code":"200"}' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-49 >> "{"@version":"1","@timestamp":"2023-07-07T14:45:47.511Z","http":{"path":"/graphql","method":"GET","host":"cantabular-server-publishing.internal.sandbox","scheme":"http","query":null,"port":14010,"status_code":"200"},"aws":{"elb":{"type":"http","action_executed":"forward","classification_reason":null,"redirect_url":null,"matched_rule_priority":"0","trace_id":"Root=1-64a8251b-2693bb36488aca4f1323537b","classification":null,"chosen_cert":{"arn":null},"error":{"reason":null},"target_group":{"arn":"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/cant-svr-pub-meta-http-sandbox/13a64716eecb64b1"}}},"elb":{"client":{"ip":"10.30.139.85","user_agent":{"raw":"Go-http-client/1.1"},"port":47976},"duration":{"backend":0.001,"response":0.0,"request":0.0},"bytes_sent":733,"load_balancer":"app/cantabular-server-pub-sandbox/b413eb65a1070c92","backend":{"ip":"10.30.137.38","status_code":"200","port":14010},"http_version":"HTTP/1.1","bytes_received":175,"tls":{"protocol":null,"cipher":null}}}[\n]"
        return 151
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[{"message":"invalid response from downstream service - should be: 200, got: 503, path: /health","stack_trace":[{"function":"github.com/ONSdigital/log.go/v2/log.Error",' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-60 >> "{"@version":"1","@timestamp":"2023-07-07T14:50:58.198Z","errors":[{"message":"invalid response from downstream service - should be: 200, got: 503, path: /health","stack_trace":[{"function":"github.com/ONSdigital/log.go/v2/log.Error","file":"/go/pkg/mod/github.com/!o!n!sdigital/log.go/v2@v2.4.1/log/log.go","line":93},{"function":"github.com/ONSdigital/dp-api-clients-go/v2/health.(*Client).Checker","file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-api-clients-go/v2@v2.252.1/health/health.go","line":90},{"function":"github.com/ONSdigital/dp-frontend-release-calendar/handlers.(*BabbageClient).Checker","file":"/tmp/build/80754af9/dp-frontend-release-calendar/handlers/babbage.go","line":42},{"function":"github.com/ONSdigital/dp-healthcheck/healthcheck.(*ticker).runCheck","file":"/go/pkg/mod/github.com/!o!n!sdigital/dp-healthcheck@v1.6.1/healthcheck/ticker.go","line":56},{"function":"runtime.goexit","file":"/usr/local/go/src/runtime/asm_amd64.s","line":1598}]}],"created_at":"2023-07-07T14:50:58.198225038Z","namespace":"dp-frontend-release-calendar","nomad":{"allocation_id":"e461d54f-493a-1c23-d2f1-b45bf2d937c1","host":"ip-10-30-141-96"},"data":"{\n  \"service\": \"Babbage\"\n}","tags":[],"severity":1,"event":"failed to request service health"}[\n]"
        return 152
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'syslog-' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-77 >> "{"message":"[2023-07-07T14:21:53,722][DEBUG][org.apache.http.wire     ] http-outgoing-13 << \"{\"took\":7,\"errors\":false,\"items\":[{\"index\":{\"_index\":\"syslog-2023-07-07\",\"_type\":\"_doc\",\"_id\":\"1007163022\",\"_version\":1,\"result\":\"created\",\"_shards\":{\"total\":2,\"successful\":2,\"failed\":0},\"_seq_no\":2254224,\"_primary_term\":1,\"status\":201}}]}\"","@version":"1","@timestamp":"2023-07-07T14:49:41.448Z","host":{"name":"ip-10-30-142-35"},"service":{"type":"logstash"},"ecs":{"version":"1.7.0"},"log":{"file":{"path":"/var/log/logstash/logstash-plain.log"},"offset":15427423},"agent":{"hostname":"ip-10-30-142-35","type":"filebeat","name":"ip-10-30-142-35","version":"7.11.2","ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"event":{"module":"logstash","dataset":"logstash.log","timezone":"+00:00"},"fileset":{"name":"log"}}[\n]"
        return 153
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"message":"INFO log/harvester.go:' in line and ' Harvester started for file: /var/lib/nomad/alloc/' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-7 >> "{"syslog_severity_code":5,"message":"INFO log/harvester.go:302 Harvester started for file: /var/lib/nomad/alloc/c6b2dc09-dbb1-d580-8963-4c5a588d4b6f/alloc/logs/dp-frontend-homepage-controller-web.stdout.9","@version":"1","@timestamp":"2023-07-07T14:47:45.000Z","syslog_timestamp":"Jul  7 14:47:45","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":86213971},"syslog_severity":"notice","received_at":"2023-07-07T14:47:46.128Z","agent":{"hostname":"ip-10-30-141-96","type":"filebeat","name":"ip-10-30-141-96","version":"7.11.2","ephemeral_id":"c480881b-1b2f-4ed6-ab95-61985140fb53","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-141-96\"}","syslog_pid":"9862","syslog_hostname":"ip-10-30-141-96","syslog_facility":"user-level","syslog_program":"filebeat"}[\n]"
        return 154
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  agent.fsm: snapshot created: duration=' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-3 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:48:57.221Z [INFO]  agent.fsm: snapshot created: duration=116.726[0xc2][0xb5]s","@version":"1","@timestamp":"2023-07-07T14:48:57.000Z","syslog_timestamp":"Jul  7 14:48:57","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":2104074},"syslog_severity":"notice","received_at":"2023-07-07T14:49:00.532Z","agent":{"hostname":"ip-10-30-142-39","type":"filebeat","name":"ip-10-30-142-39","version":"7.11.2","ephemeral_id":"38205b81-9908-4f4e-a0a1-dee0a39b2da9","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-142-39\"}","syslog_pid":"933120","syslog_hostname":"ip-10-30-142-39","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 155
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  agent.server.raft: starting snapshot up to: index=' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-3 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:48:57.221Z [INFO]  agent.server.raft: starting snapshot up to: index=58202611","@version":"1","@timestamp":"2023-07-07T14:48:57.000Z","syslog_timestamp":"Jul  7 14:48:57","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":2104208},"syslog_severity":"notice","received_at":"2023-07-07T14:49:00.532Z","agent":{"hostname":"ip-10-30-142-39","type":"filebeat","name":"ip-10-30-142-39","version":"7.11.2","ephemeral_id":"38205b81-9908-4f4e-a0a1-dee0a39b2da9","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-142-39\"}","syslog_pid":"933120","syslog_hostname":"ip-10-30-142-39","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 156
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  agent.server.snapshot: creating new snapshot: path=' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-3 >> "{"syslog_severity_code":5,"message":"    2023-07-07T14:48:57.221Z [INFO]  agent.server.snapshot: creating new snapshot: path=/var/lib/data/consul/raft/snapshots/13484-58202611-1688741337221.tmp","@version":"1","@timestamp":"2023-07-07T14:48:57.000Z","syslog_timestamp":"Jul  7 14:48:57","ecs":{"version":"1.6.0"},"syslog_facility_code":1,"log":{"file":{"path":"/var/log/syslog"},"offset":2104352},"syslog_severity":"notice","received_at":"2023-07-07T14:49:00.532Z","agent":{"hostname":"ip-10-30-142-39","type":"filebeat","name":"ip-10-30-142-39","version":"7.11.2","ephemeral_id":"38205b81-9908-4f4e-a0a1-dee0a39b2da9","id":"c81ff488-13bd-4915-83df-f9dd60fa5444"},"fields":{"type":"system"},"tags":["beats_input_codec_plain_applied"],"input":{"type":"log"},"received_from":"{\"name\":\"ip-10-30-142-39\"}","syslog_pid":"933120","syslog_hostname":"ip-10-30-142-39","syslog_facility":"user-level","syslog_program":"consul"}[\n]"
        return 157
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO] (runner) rendered' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-11 >> "{"syslog_facility":"user-level","input":{"type":"log"},"syslog_severity_code":5,"syslog_timestamp":"Jul  7 14:48:35","syslog_program":"consul-template","syslog_severity":"notice","@version":"1","syslog_pid":"11295","tags":["beats_input_codec_plain_applied"],"syslog_hostname":"ip-10-30-138-41","received_at":"2023-07-07T14:48:35.443Z","ecs":{"version":"1.6.0"},"agent":{"ephemeral_id":"87f5ab17-ed70-4e4e-b649-6921642da187","name":"ip-10-30-138-41","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-138-41"},"fields":{"type":"system"},"syslog_facility_code":1,"received_from":"{\"name\":\"ip-10-30-138-41\"}","@timestamp":"2023-07-07T14:48:35.000Z","log":{"offset":45363945,"file":{"path":"/var/log/syslog"}},"message":"2023/07/07 14:48:35.044952 [INFO] (runner) rendered \"/var/lib/consul-template/publishing-nginx.http.conf.tpl\" => \"/etc/nginx/conf.d/publishing-nginx.http.conf\""}[\n]"
        return 158
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO] (runner) executing command' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-11 >> "{"syslog_facility":"user-level","input":{"type":"log"},"syslog_severity_code":5,"syslog_timestamp":"Jul  7 14:48:35","syslog_program":"consul-template","syslog_severity":"notice","@version":"1","syslog_pid":"11295","tags":["beats_input_codec_plain_applied"],"syslog_hostname":"ip-10-30-138-41","received_at":"2023-07-07T14:48:35.443Z","ecs":{"version":"1.6.0"},"agent":{"ephemeral_id":"87f5ab17-ed70-4e4e-b649-6921642da187","name":"ip-10-30-138-41","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-138-41"},"fields":{"type":"system"},"syslog_facility_code":1,"received_from":"{\"name\":\"ip-10-30-138-41\"}","@timestamp":"2023-07-07T14:48:35.000Z","log":{"offset":45364343,"file":{"path":"/var/log/syslog"}},"message":"2023/07/07 14:48:35.045624 [INFO] (runner) executing command \"/bin/systemctl reload nginx\" from \"/var/lib/consul-template/publishing-nginx.http.conf.tpl\" => \"/etc/nginx/conf.d/publishing-nginx.http.conf\""}[\n]"
        return 159
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"/usr/local/lib/python3.9/site-packages/elasticsearch2/connection/base.py' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-59 >> "{"nomad":{"allocation_id":"61b8e259-e62a-e785-a95c-e50f1e382a82","host":"ip-10-30-140-69"},"@version":"1","tags":[],"fields":{"type":"application"},"@timestamp":"2023-07-07T14:48:36.860Z","message":"  File \"/usr/local/lib/python3.9/site-packages/elasticsearch2/connection/base.py\", line 114, in _raise_error"}[\n]"
        return 160
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"GET' in line and '"/dataset' in line and '"api.dp.aws.onsdigital.uk' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-77 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"ecs":{"version":"1.7.0"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","version":"7.11.2","type":"filebeat","hostname":"ip-10-30-142-35"},"@timestamp":"2023-07-07T14:47:54.350Z","log":{"offset":5914055,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T14:35:47,782][DEBUG][org.apache.http.wire     ] http-outgoing-64 >> \"{\"message\":\"[2023-07-07T13:40:50,679][DEBUG][org.apache.http.wire     ] http-outgoing-7 >> \\\"{\\\"@version\\\":\\\"1\\\",\\\"elb\\\":{\\\"load_balancer\\\":\\\"app/web-sandbox/82af646be10141fd\\\",\\\"duration\\\":{\\\"response\\\":0.0,\\\"backend\\\":0.011,\\\"request\\\":0.004},\\\"tls\\\":{\\\"cipher\\\":\\\"ECDHE-RSA-AES128-GCM-SHA256\\\",\\\"protocol\\\":\\\"TLSv1.2\\\"},\\\"bytes_received\\\":49,\\\"bytes_sent\\\":21071,\\\"client\\\":{\\\"port\\\":7383,\\\"user_agent\\\":{\\\"raw\\\":\\\"Blackbox Exporter/0.20.0\\\"},\\\"ip\\\":\\\"18.170.171.177\\\"},\\\"http_version\\\":\\\"HTTP/2.0\\\",\\\"backend\\\":{\\\"port\\\":80,\\\"ip\\\":\\\"10.30.140.91\\\",\\\"status_code\\\":\\\"200\\\"}},\\\"http\\\":{\\\"method\\\":\\\"GET\\\",\\\"port\\\":443,\\\"query\\\":null,\\\"path\\\":\\\"/dataset\\\",\\\"scheme\\\":\\\"https\\\",\\\"host\\\":\\\"api.dp.aws.onsdigital.uk\\\",\\\"status_code\\\":\\\"200\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:37:49.577Z\\\",\\\"aws\\\":{\\\"elb\\\":{\\\"classification_reason\\\":null,\\\"trace_id\\\":\\\"Root=1-64a8152d-7b04604e7a85218f7edead6c\\\",\\\"redirect_url\\\":null,\\\"target_group\\\":{\\\"arn\\\":\\\"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26\\\"},\\\"error\\\":{\\\"reason\\\":null},\\\"chosen_cert\\\":{\\\"arn\\\":\\\"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5\\\"},\\\"type\\\":\\\"h2\\\",\\\"classification\\\":null,\\\"action_executed\\\":\\\"forward\\\",\\\"matched_rule_priority\\\":\\\"0\\\"}}}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"hostname\":\"ip-10-30-142-35\",\"type\":\"filebeat\",\"version\":\"7.11.2\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:35:47.572Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":46740516,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 161
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"GET' in line and '"/search' in line and '"api.dp.aws.onsdigital.uk' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-77 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"ecs":{"version":"1.7.0"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"@timestamp":"2023-07-07T14:47:58.965Z","log":{"offset":5949055,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T14:35:47,783][DEBUG][org.apache.http.wire     ] http-outgoing-65 >> \"{\"message\":\"[2023-07-07T13:40:50,679][DEBUG][org.apache.http.wire     ] http-outgoing-7 >> \\\"{\\\"@version\\\":\\\"1\\\",\\\"elb\\\":{\\\"load_balancer\\\":\\\"app/web-sandbox/82af646be10141fd\\\",\\\"duration\\\":{\\\"response\\\":0.0,\\\"backend\\\":0.009,\\\"request\\\":0.001},\\\"tls\\\":{\\\"cipher\\\":\\\"ECDHE-RSA-AES128-GCM-SHA256\\\",\\\"protocol\\\":\\\"TLSv1.2\\\"},\\\"bytes_received\\\":54,\\\"bytes_sent\\\":32037,\\\"client\\\":{\\\"port\\\":1875,\\\"user_agent\\\":{\\\"raw\\\":\\\"Blackbox Exporter/0.20.0\\\"},\\\"ip\\\":\\\"18.170.171.177\\\"},\\\"http_version\\\":\\\"HTTP/2.0\\\",\\\"backend\\\":{\\\"port\\\":80,\\\"ip\\\":\\\"10.30.140.69\\\",\\\"status_code\\\":\\\"200\\\"}},\\\"http\\\":{\\\"method\\\":\\\"GET\\\",\\\"port\\\":443,\\\"query\\\":\\\"q=test\\\",\\\"path\\\":\\\"/search\\\",\\\"scheme\\\":\\\"https\\\",\\\"host\\\":\\\"api.dp.aws.onsdigital.uk\\\",\\\"status_code\\\":\\\"200\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:37:50.806Z\\\",\\\"aws\\\":{\\\"elb\\\":{\\\"classification_reason\\\":null,\\\"trace_id\\\":\\\"Root=1-64a8152e-2b99b7393a4a7ac35373d1fd\\\",\\\"redirect_url\\\":null,\\\"target_group\\\":{\\\"arn\\\":\\\"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26\\\"},\\\"error\\\":{\\\"reason\\\":null},\\\"chosen_cert\\\":{\\\"arn\\\":\\\"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5\\\"},\\\"type\\\":\\\"h2\\\",\\\"classification\\\":null,\\\"action_executed\\\":\\\"forward\\\",\\\"matched_rule_priority\\\":\\\"0\\\"}}}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"hostname\":\"ip-10-30-142-35\",\"type\":\"filebeat\",\"version\":\"7.11.2\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:35:47.666Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":46741842,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 162
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"StatusHandler: failed to ping website' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-83 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"ecs":{"version":"1.7.0"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"@timestamp":"2023-07-07T14:48:01.339Z","log":{"offset":45385175,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T13:40:24,101][DEBUG][org.apache.http.wire     ] http-outgoing-30 >> \"{\"message\":\"[2023-07-07T13:38:49,168][DEBUG][org.apache.http.wire     ] http-outgoing-6 >> \\\"{\\\"event\\\":\\\"StatusHandler: failed to ping website\\\",\\\"namespace\\\":\\\"dp-apipoc-server\\\",\\\"@version\\\":\\\"1\\\",\\\"created_at\\\":\\\"2023-03-02T13:32:05.083877329Z\\\",\\\"severity\\\":1,\\\"errors\\\":[{\\\"message\\\":\\\"Get \\\\\\\"https://www.ons.gov.uk\\\\\\\": dial tcp: lookup www.ons.gov.uk on 10.30.136.2:53: no such host\\\",\\\"stack_trace\\\":[{\\\"file\\\":\\\"/go/pkg/mod/github.com/!o!n!sdigital/log.go/v2@v2.0.9/log/log.go\\\",\\\"line\\\":92,\\\"function\\\":\\\"github.com/ONSdigital/log.go/v2/log.Error\\\"},{\\\"file\\\":\\\"/tmp/build/80754af9/dp-apipoc-server/handler/common.go\\\",\\\"line\\\":34,\\\"function\\\":\\\"github.com/ONSdigital/dp-apipoc-server/handler.logOut\\\"},{\\\"file\\\":\\\"/tmp/build/80754af9/dp-apipoc-server/handler/ops.go\\\",\\\"line\\\":31,\\\"function\\\":\\\"github.com/ONSdigital/dp-apipoc-server/handler.OpsHandler.StatusHandler\\\"},{\\\"file\\\":\\\"/usr/local/go/src/net/http/server.go\\\",\\\"line\\\":2109,\\\"function\\\":\\\"net/http.HandlerFunc.ServeHTTP\\\"},{\\\"file\\\":\\\"/go/pkg/mod/github.com/gorilla/mux@v1.8.0/mux.go\\\",\\\"line\\\":210,\\\"function\\\":\\\"github.com/gorilla/mux.(*Router).ServeHTTP\\\"},{\\\"file\\\":\\\"/go/pkg/mod/github.com/rs/cors@v1.8.0/cors.go\\\",\\\"line\\\":219,\\\"function\\\":\\\"github.com/rs/cors.(*Cors).Handler.func1\\\"},{\\\"file\\\":\\\"/usr/local/go/src/net/http/server.go\\\",\\\"line\\\":2109,\\\"function\\\":\\\"net/http.HandlerFunc.ServeHTTP\\\"},{\\\"file\\\":\\\"/tmp/build/80754af9/dp-apipoc-server/routing/routes.go\\\",\\\"line\\\":61,\\\"function\\\":\\\"github.com/ONSdigital/dp-apipoc-server/routing.DeprecationMiddleware.func1.1\\\"},{\\\"file\\\":\\\"/usr/local/go/src/net/http/server.go\\\",\\\"line\\\":2109,\\\"function\\\":\\\"net/http.HandlerFunc.ServeHTTP\\\"},{\\\"file\\\":\\\"/go/pkg/mod/github.com/!o!n!sdigital/dp-net/v2@v2.2.0-beta/request/request.go\\\",\\\"line\\\":61,\\\"function\\\":\\\"github.com/ONSdigital/dp-net/v2/request.HandlerRequestID.func1.1\\\"}]}],\\\"tags\\\":[],\\\"@timestamp\\\":\\\"2023-03-02T13:32:05.083Z\\\",\\\"nomad\\\":{\\\"allocation_id\\\":\\\"7907d1cb-c24d-16df-2323-edbf9cd32fa0\\\",\\\"host\\\":\\\"ip-10-30-140-223\\\"},\\\"data\\\":null}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:40:23.793Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":819003,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 163
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"GET' in line and '"/' in line and '"dp.aws.onsdigital.uk' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-77 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"ecs":{"version":"1.7.0"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"@timestamp":"2023-07-07T14:48:22.084Z","log":{"offset":6054822,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T14:35:47,813][DEBUG][org.apache.http.wire     ] http-outgoing-66 >> \"{\"message\":\"[2023-07-07T13:40:50,679][DEBUG][org.apache.http.wire     ] http-outgoing-7 >> \\\"{\\\"@version\\\":\\\"1\\\",\\\"elb\\\":{\\\"load_balancer\\\":\\\"app/web-sandbox/82af646be10141fd\\\",\\\"duration\\\":{\\\"response\\\":0.0,\\\"backend\\\":0.008,\\\"request\\\":0.0},\\\"tls\\\":{\\\"cipher\\\":\\\"ECDHE-RSA-AES128-GCM-SHA256\\\",\\\"protocol\\\":\\\"TLSv1.2\\\"},\\\"bytes_received\\\":40,\\\"bytes_sent\\\":83168,\\\"client\\\":{\\\"port\\\":15465,\\\"user_agent\\\":{\\\"raw\\\":\\\"Blackbox Exporter/0.20.0\\\"},\\\"ip\\\":\\\"3.9.123.111\\\"},\\\"http_version\\\":\\\"HTTP/2.0\\\",\\\"backend\\\":{\\\"port\\\":80,\\\"ip\\\":\\\"10.30.140.225\\\",\\\"status_code\\\":\\\"200\\\"}},\\\"http\\\":{\\\"method\\\":\\\"GET\\\",\\\"port\\\":443,\\\"query\\\":null,\\\"path\\\":\\\"/\\\",\\\"scheme\\\":\\\"https\\\",\\\"host\\\":\\\"dp.aws.onsdigital.uk\\\",\\\"status_code\\\":\\\"200\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:38:20.119Z\\\",\\\"aws\\\":{\\\"elb\\\":{\\\"classification_reason\\\":null,\\\"trace_id\\\":\\\"Root=1-64a8154c-53fde03f4a413c3a7a121f36\\\",\\\"redirect_url\\\":null,\\\"target_group\\\":{\\\"arn\\\":\\\"arn:aws:elasticloadbalancing:eu-west-2:337289154253:targetgroup/web-sandbox/01fb4d69d438dd26\\\"},\\\"error\\\":{\\\"reason\\\":null},\\\"chosen_cert\\\":{\\\"arn\\\":\\\"arn:aws:acm:eu-west-2:337289154253:certificate/ead326d2-2832-485e-8747-a9040a5dfff5\\\"},\\\"type\\\":\\\"h2\\\",\\\"classification\\\":null,\\\"action_executed\\\":\\\"forward\\\",\\\"matched_rule_priority\\\":\\\"0\\\"}}}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:35:47.667Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":46743171,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 163
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'retryer: send unwait signal to consumer' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-82 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","version":"7.11.2","type":"filebeat","hostname":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"@timestamp":"2023-07-07T14:49:20.584Z","log":{"offset":73486559,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T13:30:31,344][DEBUG][org.apache.http.wire     ] http-outgoing-25 >> \"{\"message\":\"[2023-07-07T13:13:24,609][DEBUG][org.apache.http.wire     ] http-outgoing-22 >> \\\"{\\\"message\\\":\\\"[2023-07-07T13:11:05,852][DEBUG][org.apache.http.wire     ] http-outgoing-18 >> \\\\\\\"{\\\\\\\"syslog_timestamp\\\\\\\":\\\\\\\"Jul  7 13:10:25\\\\\\\",\\\\\\\"message\\\\\\\":\\\\\\\"INFO [publisher] pipeline/retry.go:219 retryer: send unwait signal to consumer\\\\\\\",\\\\\\\"syslog_hostname\\\\\\\":\\\\\\\"ip-10-30-142-164\\\\\\\",\\\\\\\"syslog_program\\\\\\\":\\\\\\\"filebeat\\\\\\\",\\\\\\\"@version\\\\\\\":\\\\\\\"1\\\\\\\",\\\\\\\"received_at\\\\\\\":\\\\\\\"2023-07-07T13:10:25.925Z\\\\\\\",\\\\\\\"ecs\\\\\\\":{\\\\\\\"version\\\\\\\":\\\\\\\"1.6.0\\\\\\\"},\\\\\\\"agent\\\\\\\":{\\\\\\\"name\\\\\\\":\\\\\\\"ip-10-30-142-164\\\\\\\",\\\\\\\"ephemeral_id\\\\\\\":\\\\\\\"77721878-8c31-4461-8b5d-32c2c29318c4\\\\\\\",\\\\\\\"id\\\\\\\":\\\\\\\"c81ff488-13bd-4915-83df-f9dd60fa5444\\\\\\\",\\\\\\\"type\\\\\\\":\\\\\\\"filebeat\\\\\\\",\\\\\\\"version\\\\\\\":\\\\\\\"7.11.2\\\\\\\",\\\\\\\"hostname\\\\\\\":\\\\\\\"ip-10-30-142-164\\\\\\\"},\\\\\\\"received_from\\\\\\\":\\\\\\\"{\\\\\\\\\\\\\\\"name\\\\\\\\\\\\\\\":\\\\\\\\\\\\\\\"ip-10-30-142-164\\\\\\\\\\\\\\\"}\\\\\\\",\\\\\\\"fields\\\\\\\":{\\\\\\\"type\\\\\\\":\\\\\\\"system\\\\\\\"},\\\\\\\"syslog_pid\\\\\\\":\\\\\\\"511\\\\\\\",\\\\\\\"syslog_severity\\\\\\\":\\\\\\\"notice\\\\\\\",\\\\\\\"syslog_facility\\\\\\\":\\\\\\\"user-level\\\\\\\",\\\\\\\"tags\\\\\\\":[\\\\\\\"beats_input_codec_plain_applied\\\\\\\"],\\\\\\\"syslog_facility_code\\\\\\\":1,\\\\\\\"input\\\\\\\":{\\\\\\\"type\\\\\\\":\\\\\\\"log\\\\\\\"},\\\\\\\"@timestamp\\\\\\\":\\\\\\\"2023-07-07T13:10:25.000Z\\\\\\\",\\\\\\\"log\\\\\\\":{\\\\\\\"offset\\\\\\\":3801983,\\\\\\\"file\\\\\\\":{\\\\\\\"path\\\\\\\":\\\\\\\"/var/log/syslog\\\\\\\"}},\\\\\\\"syslog_severity_code\\\\\\\":5}[\\\\\\\\n]\\\\\\\"\\\",\\\"event\\\":{\\\"dataset\\\":\\\"logstash.log\\\",\\\"timezone\\\":\\\"+00:00\\\",\\\"module\\\":\\\"logstash\\\"},\\\"@version\\\":\\\"1\\\",\\\"ecs\\\":{\\\"version\\\":\\\"1.7.0\\\"},\\\"agent\\\":{\\\"name\\\":\\\"ip-10-30-142-35\\\",\\\"ephemeral_id\\\":\\\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\\\",\\\"id\\\":\\\"c81ff488-13bd-4915-83df-f9dd60fa5444\\\",\\\"hostname\\\":\\\"ip-10-30-142-35\\\",\\\"type\\\":\\\"filebeat\\\",\\\"version\\\":\\\"7.11.2\\\"},\\\"fileset\\\":{\\\"name\\\":\\\"log\\\"},\\\"tags\\\":[\\\"beats_input_codec_plain_applied\\\"],\\\"input\\\":{\\\"type\\\":\\\"log\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:13:24.143Z\\\",\\\"host\\\":{\\\"name\\\":\\\"ip-10-30-142-35\\\"},\\\"log\\\":{\\\"offset\\\":50008837,\\\"file\\\":{\\\"path\\\":\\\"/var/log/logstash/logstash-plain.log\\\"}},\\\"service\\\":{\\\"type\\\":\\\"logstash\\\"}}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\",\"type\":\"filebeat\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:30:31.192Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":100479019,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 164
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[ERROR] agent.server.connect: Failed to initialize Connect CA: routine=' in line and '"CA initialization' in line and '" error=' in line and '"error generating CA root certificate: Error making API request.' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-82 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"ecs":{"version":"1.7.0"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"@timestamp":"2023-07-07T14:50:32.086Z","log":{"offset":9974975,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T14:29:37,051][DEBUG][org.apache.http.wire     ] http-outgoing-3 >> \"{\"syslog_timestamp\":\"Jul  7 14:29:35\",\"message\":\"    2023-07-07T14:29:35.063Z [ERROR] agent.server.connect: Failed to initialize Connect CA: routine=\\\"CA initialization\\\" error=\\\"error generating CA root certificate: Error making API request.\",\"syslog_hostname\":\"ip-10-30-142-190\",\"syslog_program\":\"consul\",\"@version\":\"1\",\"received_at\":\"2023-07-07T14:29:35.975Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-142-190\",\"ephemeral_id\":\"03715cb6-7b08-465f-a290-6519f3108c43\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-190\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-142-190\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"938023\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:29:35.000Z\",\"log\":{\"offset\":2441779,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\""}[\n]"
        return 165
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'com.github.onsdigital.zebedee.exceptions.NotFoundException: Could not find requested content, path:file:' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-83 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"@timestamp":"2023-07-07T14:50:32.074Z","log":{"offset":28013555,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T14:03:27,797][DEBUG][org.apache.http.wire     ] http-outgoing-43 >> \"{\"message\":\"[2023-07-07T13:38:35,091][DEBUG][org.apache.http.wire     ] http-outgoing-7 >> \\\"{\\\"event\\\":\\\"Zebedee Reader API error\\\",\\\"namespace\\\":\\\"zebedee-reader\\\",\\\"@version\\\":\\\"1\\\",\\\"created_at\\\":\\\"2023-07-07T10:24:32.488Z\\\",\\\"severity\\\":1,\\\"errors\\\":[{\\\"message\\\":\\\"com.github.onsdigital.zebedee.exceptions.NotFoundException: Could not find requested content, path:file:///content/releases/test27oct/data_cy.json\\\",\\\"stack_trace\\\":[{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.FileSystemContentReader\\\",\\\"line\\\":343,\\\"function\\\":\\\"assertExists\\\"},{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.FileSystemContentReader\\\",\\\"line\\\":189,\\\"function\\\":\\\"getContentLength\\\"},{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.ZebedeeReader\\\",\\\"line\\\":178,\\\"function\\\":\\\"getPublishedContentLength\\\"},{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.api.ReadRequestHandler\\\",\\\"line\\\":140,\\\"function\\\":\\\"lambda$getContentLength$7\\\"},{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.api.ReadRequestHandler\\\",\\\"line\\\":177,\\\"function\\\":\\\"get\\\"},{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.api.ReadRequestHandler\\\",\\\"line\\\":185,\\\"function\\\":\\\"get\\\"},{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.api.ReadRequestHandler\\\",\\\"line\\\":138,\\\"function\\\":\\\"getContentLength\\\"},{\\\"file\\\":\\\"com.github.onsdigital.zebedee.reader.api.endpoint.FileSize\\\",\\\"line\\\":31,\\\"function\\\":\\\"get\\\"},{\\\"file\\\":\\\"sun.reflect.GeneratedMethodAccessor28\\\",\\\"line\\\":-1,\\\"function\\\":\\\"invoke\\\"},{\\\"file\\\":\\\"sun.reflect.DelegatingMethodAccessorImpl\\\",\\\"line\\\":43,\\\"function\\\":\\\"invoke\\\"},{\\\"file\\\":\\\"java.lang.reflect.Method\\\",\\\"line\\\":498,\\\"function\\\":\\\"invoke\\\"},{\\\"file\\\":\\\"com.github.davidcarboni.restolino.api.Router\\\",\\\"line\\\":452,\\\"function\\\":\\\"invoke\\\"},{\\\"file\\\":\\\"com.github.davidcarboni.restolino.api.Router\\\",\\\"line\\\":356,\\\"function\\\":\\\"handleRequest\\\"},{\\\"file\\\":\\\"com.github.davidcarboni.restolino.api.Router\\\",\\\"line\\\":331,\\\"function\\\":\\\"doMethod\\\"},{\\\"file\\\":\\\"com.github.davidcarboni.restolino.api.Router\\\",\\\"line\\\":191,\\\"function\\\":\\\"get\\\"},{\\\"file\\\":\\\"com.github.davidcarboni.restolino.jetty.ApiHandler\\\",\\\"line\\\":39,\\\"function\\\":\\\"handle\\\"},{\\\"file\\\":\\\"com.github.davidcarboni.restolino.jetty.MainHandler\\\",\\\"line\\\":142,\\\"function\\\":\\\"handle\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.handler.gzip.GzipHandler\\\",\\\"line\\\":763,\\\"function\\\":\\\"handle\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.handler.StatisticsHandler\\\",\\\"line\\\":179,\\\"function\\\":\\\"handle\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.handler.HandlerWrapper\\\",\\\"line\\\":127,\\\"function\\\":\\\"handle\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.Server\\\",\\\"line\\\":516,\\\"function\\\":\\\"handle\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.HttpChannel\\\",\\\"line\\\":388,\\\"function\\\":\\\"lambda$handle$1\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.HttpChannel\\\",\\\"line\\\":633,\\\"function\\\":\\\"dispatch\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.HttpChannel\\\",\\\"line\\\":380,\\\"function\\\":\\\"handle\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.server.HttpConnection\\\",\\\"line\\\":279,\\\"function\\\":\\\"onFillable\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.io.AbstractConnection$ReadCallback\\\",\\\"line\\\":311,\\\"function\\\":\\\"succeeded\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.io.FillInterest\\\",\\\"line\\\":105,\\\"function\\\":\\\"fillable\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.io.ChannelEndPoint$1\\\",\\\"line\\\":104,\\\"function\\\":\\\"run\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.util.thread.strategy.EatWhatYouKill\\\",\\\"line\\\":336,\\\"function\\\":\\\"runTask\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.util.thread.strategy.EatWhatYouKill\\\",\\\"line\\\":313,\\\"function\\\":\\\"doProduce\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.util.thread.strategy.EatWhatYouKill\\\",\\\"line\\\":171,\\\"function\\\":\\\"tryProduce\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.util.thread.strategy.EatWhatYouKill\\\",\\\"line\\\":129,\\\"function\\\":\\\"run\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.util.thread.ReservedThreadExecutor$ReservedThread\\\",\\\"line\\\":383,\\\"function\\\":\\\"run\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.util.thread.QueuedThreadPool\\\",\\\"line\\\":882,\\\"function\\\":\\\"runJob\\\"},{\\\"file\\\":\\\"org.eclipse.jetty.util.thread.QueuedThreadPool$Runner\\\",\\\"line\\\":1036,\\\"function\\\":\\\"run\\\"},{\\\"file\\\":\\\"java.lang.Thread\\\",\\\"line\\\":750,\\\"function\\\":\\\"run\\\"}]}],\\\"trace_id\\\":\\\"fxXAgwPOuCnraNxoHtFl\\\",\\\"tags\\\":[],\\\"@timestamp\\\":\\\"2023-07-07T10:24:32.488Z\\\",\\\"nomad\\\":{\\\"allocation_id\\\":\\\"a4b4f00b-c27a-23d4-e4f9-7780fe7fae62\\\",\\\"host\\\":\\\"ip-10-30-140-12\\\"},\\\"data\\\":\\\"{\\\\n  \\\\\\\"exception_status_code\\\\\\\": 404\\\\n}\\\"}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\",\"type\":\"filebeat\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:03:26.932Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":24503640,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 166
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"message' in line and '* 1 error occurred:' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-82 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"@timestamp":"2023-07-07T14:50:32.226Z","log":{"offset":9978332,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T14:29:37,051][DEBUG][org.apache.http.wire     ] http-outgoing-3 >> \"{\"syslog_timestamp\":\"Jul  7 14:29:35\",\"message\":\"* 1 error occurred:\",\"syslog_hostname\":\"ip-10-30-142-190\",\"syslog_program\":\"consul\",\"@version\":\"1\",\"received_at\":\"2023-07-07T14:29:35.975Z\",\"ecs\":{\"version\":\"1.6.0\"},\"agent\":{\"name\":\"ip-10-30-142-190\",\"ephemeral_id\":\"03715cb6-7b08-465f-a290-6519f3108c43\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-190\"},\"received_from\":\"{\\\"name\\\":\\\"ip-10-30-142-190\\\"}\",\"fields\":{\"type\":\"system\"},\"syslog_pid\":\"938023\",\"syslog_severity\":\"notice\",\"syslog_facility\":\"user-level\",\"tags\":[\"beats_input_codec_plain_applied\"],\"syslog_facility_code\":1,\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T14:29:35.000Z\",\"log\":{\"offset\":2442190,\"file\":{\"path\":\"/var/log/syslog\"}},\"syslog_severity_code\":5}[\\n]\""}[\n]"
        return 167
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and 'got error while decoding json' in line and 'unexpected EOF' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-82 >> "{"input":{"type":"log"},"service":{"type":"logstash"},"event":{"dataset":"logstash.log","module":"logstash","timezone":"+00:00"},"host":{"name":"ip-10-30-142-35"},"@version":"1","tags":["beats_input_codec_plain_applied"],"fileset":{"name":"log"},"agent":{"ephemeral_id":"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b","name":"ip-10-30-142-35","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-35"},"ecs":{"version":"1.7.0"},"@timestamp":"2023-07-07T14:50:48.338Z","log":{"offset":58379204,"file":{"path":"/var/log/logstash/logstash-plain.log"}},"message":"[2023-07-07T13:36:52,731][DEBUG][org.apache.http.wire     ] http-outgoing-29 >> \"{\"message\":\"[2023-07-07T13:22:10,943][DEBUG][org.apache.http.wire     ] http-outgoing-11 >> \\\"{\\\"syslog_timestamp\\\":\\\"Jul  7 13:22:08\\\",\\\"message\\\":\\\"time=\\\\\\\"2023-07-07T13:22:08.593067639Z\\\\\\\" level=warning msg=\\\\\\\"got error while decoding json\\\\\\\" error=\\\\\\\"unexpected EOF\\\\\\\" retries=4\\\",\\\"syslog_hostname\\\":\\\"ip-10-30-138-60\\\",\\\"syslog_program\\\":\\\"dockerd\\\",\\\"@version\\\":\\\"1\\\",\\\"received_at\\\":\\\"2023-07-07T13:22:10.066Z\\\",\\\"ecs\\\":{\\\"version\\\":\\\"1.6.0\\\"},\\\"agent\\\":{\\\"name\\\":\\\"ip-10-30-138-60\\\",\\\"ephemeral_id\\\":\\\"933a59a1-6de1-49e1-ac6c-dc3b6ffd15d4\\\",\\\"id\\\":\\\"c81ff488-13bd-4915-83df-f9dd60fa5444\\\",\\\"type\\\":\\\"filebeat\\\",\\\"version\\\":\\\"7.11.2\\\",\\\"hostname\\\":\\\"ip-10-30-138-60\\\"},\\\"received_from\\\":\\\"{\\\\\\\"name\\\\\\\":\\\\\\\"ip-10-30-138-60\\\\\\\"}\\\",\\\"fields\\\":{\\\"type\\\":\\\"system\\\"},\\\"syslog_pid\\\":\\\"12783\\\",\\\"syslog_severity\\\":\\\"notice\\\",\\\"syslog_facility\\\":\\\"user-level\\\",\\\"tags\\\":[\\\"beats_input_codec_plain_applied\\\"],\\\"syslog_facility_code\\\":1,\\\"input\\\":{\\\"type\\\":\\\"log\\\"},\\\"@timestamp\\\":\\\"2023-07-07T13:22:08.000Z\\\",\\\"log\\\":{\\\"offset\\\":41072913,\\\"file\\\":{\\\"path\\\":\\\"/var/log/syslog\\\"}},\\\"syslog_severity_code\\\":5}[\\\\n]\\\"\",\"event\":{\"dataset\":\"logstash.log\",\"timezone\":\"+00:00\",\"module\":\"logstash\"},\"@version\":\"1\",\"ecs\":{\"version\":\"1.7.0\"},\"agent\":{\"name\":\"ip-10-30-142-35\",\"ephemeral_id\":\"21762cb1-c97e-4e6c-bef6-ba4b25ce2e3b\",\"id\":\"c81ff488-13bd-4915-83df-f9dd60fa5444\",\"type\":\"filebeat\",\"version\":\"7.11.2\",\"hostname\":\"ip-10-30-142-35\"},\"fileset\":{\"name\":\"log\"},\"tags\":[\"beats_input_codec_plain_applied\"],\"input\":{\"type\":\"log\"},\"@timestamp\":\"2023-07-07T13:36:51.992Z\",\"host\":{\"name\":\"ip-10-30-142-35\"},\"log\":{\"offset\":44295526,\"file\":{\"path\":\"/var/log/logstash/logstash-plain.log\"}},\"service\":{\"type\":\"logstash\"}}[\\n]\""}[\n]"
        return 168
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '"error":"1 error occurred:' in line and '* permission denied' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-80 >> "{"syslog_facility":"user-level","input":{"type":"log"},"syslog_severity_code":5,"syslog_timestamp":"Jul  7 14:50:55","syslog_program":"vault","syslog_severity":"notice","vault":{"auth":{"display_name":"token","token_type":"service","policies":["consul-connect","default"],"accessor":"hmac-sha256:9f7a67738a617c296b8ba7eb8a2dfff33c16313d175df3580fdf213c01fbb79a","client_token":"hmac-sha256:184aeca6f5defdc523af99d3071c33f0d186877cd976bdab346d26b67d7eb873","token_policies":["consul-connect","default"]},"error":"1 error occurred:\n\t* permission denied\n\n","type":"request","time":"2023-07-07T14:50:55.180132866Z","request":{"namespace":{"id":"root"},"id":"d24ea207-f0a0-cd14-1621-b7767456bf47","client_token_accessor":"hmac-sha256:9f7a67738a617c296b8ba7eb8a2dfff33c16313d175df3580fdf213c01fbb79a","path":"connect-root/ca/pem","remote_address":"10.30.142.190","client_token":"hmac-sha256:184aeca6f5defdc523af99d3071c33f0d186877cd976bdab346d26b67d7eb873","operation":"read"}},"@version":"1","syslog_pid":"1651962","tags":["beats_input_codec_plain_applied"],"syslog_hostname":"ip-10-30-142-161","received_at":"2023-07-07T14:50:58.151Z","agent":{"ephemeral_id":"7652a579-9265-4357-bb58-d2df8251033a","name":"ip-10-30-142-161","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-161"},"ecs":{"version":"1.6.0"},"fields":{"type":"auth"},"syslog_facility_code":1,"received_from":"{\"name\":\"ip-10-30-142-161\"}","@timestamp":"2023-07-07T14:50:55.000Z","log":{"offset":16670025,"file":{"path":"/var/log/auth.log"}},"message":"{\"time\":\"2023-07-07T14:50:55.180132866Z\",\"type\":\"request\",\"auth\":{\"client_token\":\"hmac-sha256:184aeca6f5defdc523af99d3071c33f0d186877cd976bdab346d26b67d7eb873\",\"accessor\":\"hmac-sha256:9f7a67738a617c296b8ba7eb8a2dfff33c16313d175df3580fdf213c01fbb79a\",\"display_name\":\"token\",\"policies\":[\"consul-connect\",\"default\"],\"token_policies\":[\"consul-connect\",\"default\"],\"token_type\":\"service\"},\"request\":{\"id\":\"d24ea207-f0a0-cd14-1621-b7767456bf47\",\"operation\":\"read\",\"client_token\":\"hmac-sha256:184aeca6f5defdc523af99d3071c33f0d186877cd976bdab346d26b67d7eb873\",\"client_token_accessor\":\"hmac-sha256:9f7a67738a617c296b8ba7eb8a2dfff33c16313d175df3580fdf213c01fbb79a\",\"namespace\":{\"id\":\"root\"},\"path\":\"connect-root/ca/pem\",\"remote_address\":\"10.30.142.190\"},\"error\":\"1 error occurred:\\n\\t* permission denied\\n\\n\"}"}[\n]"
        return 169
    if '[DEBUG][org.apache.http.wire     ] http-outgoing-' in line and '[INFO]  connect.ca.vault: Successfully renewed token for Vault provider' in line:
        # this copes with all variants of:
        # [DEBUG][org.apache.http.wire     ] http-outgoing-3 >> "{"syslog_facility":"user-level","input":{"type":"log"},"syslog_severity_code":5,"syslog_timestamp":"Jul  7 14:50:55","syslog_program":"consul","syslog_severity":"notice","@version":"1","syslog_pid":"938023","tags":["beats_input_codec_plain_applied"],"syslog_hostname":"ip-10-30-142-190","received_at":"2023-07-07T14:50:59.044Z","agent":{"ephemeral_id":"03715cb6-7b08-465f-a290-6519f3108c43","name":"ip-10-30-142-190","id":"c81ff488-13bd-4915-83df-f9dd60fa5444","type":"filebeat","version":"7.11.2","hostname":"ip-10-30-142-190"},"ecs":{"version":"1.6.0"},"fields":{"type":"system"},"syslog_facility_code":1,"received_from":"{\"name\":\"ip-10-30-142-190\"}","@timestamp":"2023-07-07T14:50:55.000Z","log":{"offset":2500819,"file":{"path":"/var/log/syslog"}},"message":"    2023-07-07T14:50:55.179Z [INFO]  connect.ca.vault: Successfully renewed token for Vault provider"}[\n]"
        return 170
    if '[INFO ][logstash.runner          ] Log4j configuration path used is: /etc/logstash/log4j2.properties' in line:
        # this copes with all variants of:
        # WARN  Unknown error line: [2023-07-26T12:01:18,517][INFO ][logstash.runner          ] Log4j configuration path used is: /etc/logstash/log4j2.properties
        return 171
    if '[INFO ][logstash.runner          ] Starting Logstash {"logstash.version"=>"7.12.1"' in line:
        # this copes with all variants of:
        # [INFO ][logstash.runner          ] Starting Logstash {"logstash.version"=>"7.12.1"
        return 172
    if '[ERROR][logstash.agent           ] Internal API server error {:status=>500, :request_method=>"GET"' in line:
        # this copes with all variants of:
        # [ERROR][logstash.agent           ] Internal API server error {:status=>500, :request_method=>"GET"
        return 173
    if '[ERROR][logstash.agent           ] API HTTP Request {:status=>500, :request_method=>"GET", :path_info=>"/_node", :query_string=>"", :http_version=>"HTTP/1.1", :http_accept=>nil}' in line:
        # [ERROR][logstash.agent           ] API HTTP Request {:status=>500, :request_method=>"GET", :path_info=>"/_node", :query_string=>"", :http_version=>"HTTP/1.1", :http_accept=>nil}
        return 174
    if '[ERROR][logstash.agent           ] API HTTP Request {:status=>500, :request_method=>"GET", :path_info=>"/_node/stats", :query_string=>"", :http_version=>"HTTP/1.1", :http_accept=>nil}' in line:
        # [ERROR][logstash.agent           ] API HTTP Request {:status=>500, :request_method=>"GET", :path_info=>"/_node/stats", :query_string=>"", :http_version=>"HTTP/1.1", :http_accept=>nil}
        return 174
    if 'Relying on default value of `pipeline.ecs_compatibility`, which may change in a future major release of Logstash. To avoid unexpected changes when upgrading Logstash, please explicitly declare your desired ECS Compatibility mode.' in line:
        # this copes with all variants of:
        # Relying on default value of `pipeline.ecs_compatibility`, which may change in a future major release of Logstash. To avoid unexpected changes when upgrading Logstash, please explicitly declare your desired ECS Compatibility mode.
        return 175
    if '[WARN ][logstash.runner          ] SIGTERM received. Shutting down.' in line:
        # [WARN ][logstash.runner          ] SIGTERM received. Shutting down.
        return 176
    if '[INFO ][logstash.runner          ] Starting Logstash {"logstash.version"=>"7.17.12"' in line:
        # this copes with all variants of:
        # [INFO ][logstash.runner          ] Starting Logstash {"logstash.version"=>"7.17.12", "jruby.version"=>"jruby 9.2.20.1 (2.5.8) 2021-11-30 2a2962fbd1 OpenJDK 64-Bit Server VM 11.0.19+7 on 11.0.19+7 +indy +jit [linux-x86_64]"}
        return 177
    if '[INFO ][logstash.runner          ] JVM bootstrap flags:' in line:
        # this copes with all variants of:
        # [INFO ][logstash.runner          ] JVM bootstrap flags: [-Xms1g, -Xmx1g, -XX:+UseConcMarkSweepGC, -XX:CMSInitiatingOccupancyFraction=75, -XX:+UseCMSInitiatingOccupancyOnly, -Djava.awt.headless=true, -Dfile.encoding=UTF-8, -Djdk.io.File.enableADS=true, -Djruby.compile.invokedynamic=true, -Djruby.jit.threshold=0, -Djruby.regexp.interruptible=true, -XX:+HeapDumpOnOutOfMemoryError, -Djava.security.egd=file:/dev/urandom, -Dlog4j2.isThreadContextMapInheritable=true]
        return 178
    if "[INFO ][logstash.agent           ] Successfully started Logstash API endpoint {:port=>9600, :ssl_enabled=>false}" in line:
        # [INFO ][logstash.agent           ] Successfully started Logstash API endpoint {:port=>9600, :ssl_enabled=>false}
        return 179
    if '[ERROR][logstash.plugins.registry] Unable to load plugin. {:type=>"output", :name=>"amazon_es"}' in line:
        # [ERROR][logstash.plugins.registry] Unable to load plugin. {:type=>"output", :name=>"amazon_es"}
        return 180
    if '[ERROR][logstash.agent           ] Failed to execute action {:action=>LogStash::PipelineAction::Create/pipeline_id:' in line:
        # this copes with all variants of:
        # [ERROR][logstash.agent           ] Failed to execute action {:action=>LogStash::PipelineAction::Create/pipeline_id:main, :exception=>"Java::JavaLang::IllegalStateException", :message=>"Unable to configure plugins: (PluginLoadingError) Couldn't find any output plugin named 'amazon_es'. Are you sure this is correct? Trying to load the amazon_es output plugin resulted in this error: Unable to load the requested plugin named amazon_es of type output. The plugin is not installed.", :backtrace=>["org.logstash.config.ir.CompiledPipeline.<init>(CompiledPipeline.java:120)", "org.logstash.execution.JavaBasePipelineExt.initialize(JavaBasePipelineExt.java:86)", "org.logstash.execution.JavaBasePipelineExt$INVOKER$i$1$0$initialize.call(JavaBasePipelineExt$INVOKER$i$1$0$initialize.gen)", "org.jruby.internal.runtime.methods.JavaMethod$JavaMethodN.call(JavaMethod.java:837)", "org.jruby.ir.runtime.IRRuntimeHelpers.instanceSuper(IRRuntimeHelpers.java:1169)", "org.jruby.ir.runtime.IRRuntimeHelpers.instanceSuperSplatArgs(IRRuntimeHelpers.java:1156)", "org.jruby.ir.targets.InstanceSuperInvokeSite.invoke(InstanceSuperInvokeSite.java:39)", "usr.share.logstash.logstash_minus_core.lib.logstash.java_pipeline.RUBY$method$initialize$0(/usr/share/logstash/logstash-core/lib/logstash/java_pipeline.rb:48)", "org.jruby.internal.runtime.methods.CompiledIRMethod.call(CompiledIRMethod.java:80)", "org.jruby.internal.runtime.methods.MixedModeIRMethod.call(MixedModeIRMethod.java:70)", "org.jruby.runtime.callsite.CachingCallSite.cacheAndCall(CachingCallSite.java:333)", "org.jruby.runtime.callsite.CachingCallSite.call(CachingCallSite.java:87)", "org.jruby.RubyClass.newInstance(RubyClass.java:939)", "org.jruby.RubyClass$INVOKER$i$newInstance.call(RubyClass$INVOKER$i$newInstance.gen)", "org.jruby.ir.targets.InvokeSite.invoke(InvokeSite.java:207)", "usr.share.logstash.logstash_minus_core.lib.logstash.pipeline_action.create.RUBY$method$execute$0(/usr/share/logstash/logstash-core/lib/logstash/pipeline_action/create.rb:52)", "usr.share.logstash.logstash_minus_core.lib.logstash.pipeline_action.create.RUBY$method$execute$0$__VARARGS__(/usr/share/logstash/logstash-core/lib/logstash/pipeline_action/create.rb:50)", "org.jruby.internal.runtime.methods.CompiledIRMethod.call(CompiledIRMethod.java:80)", "org.jruby.internal.runtime.methods.MixedModeIRMethod.call(MixedModeIRMethod.java:70)", "org.jruby.ir.targets.InvokeSite.invoke(InvokeSite.java:207)", "usr.share.logstash.logstash_minus_core.lib.logstash.agent.RUBY$block$converge_state$2(/usr/share/logstash/logstash-core/lib/logstash/agent.rb:392)", "org.jruby.runtime.CompiledIRBlockBody.callDirect(CompiledIRBlockBody.java:138)", "org.jruby.runtime.IRBlockBody.call(IRBlockBody.java:58)", "org.jruby.runtime.IRBlockBody.call(IRBlockBody.java:52)", "org.jruby.runtime.Block.call(Block.java:139)", "org.jruby.RubyProc.call(RubyProc.java:318)", "org.jruby.internal.runtime.RubyRunnable.run(RubyRunnable.java:105)", "java.base/java.lang.Thread.run(Thread.java:829)"]}
        return 181
    if '[ERROR][logstash.agent           ] An exception happened when converging configuration {:exception=>LogStash::Error, :message=>"Don' in line:
        # this copes with all variants of:
        # [ERROR][logstash.agent           ] An exception happened when converging configuration {:exception=>LogStash::Error, :message=>"Don't know how to handle `Java::JavaLang::IllegalStateException` for `PipelineAction::Create<main>`"}
        return 182
    if '[FATAL][logstash.runner          ] An unexpected error occurred! {:error=>#<LogStash::Error: Don' in line:
        # this copes with all variants of:
        # [FATAL][logstash.runner          ] An unexpected error occurred! {:error=>#<LogStash::Error: Don't know how to handle `Java::JavaLang::IllegalStateException` for `PipelineAction::Create<main>`>, :backtrace=>["org/logstash/execution/ConvergeResultExt.java:135:in `create'", "org/logstash/execution/ConvergeResultExt.java:60:in `add'", "/usr/share/logstash/logstash-core/lib/logstash/agent.rb:405:in `block in converge_state'"]}
        return 183
    if '[FATAL][org.logstash.Logstash    ] Logstash stopped processing because of an error: (SystemExit) exit' in line:
        # [FATAL][org.logstash.Logstash    ] Logstash stopped processing because of an error: (SystemExit) exit
        return 184
    if '[WARN ][org.logstash.common.io.DeadLetterQueueWriter] Event previously submitted to dead letter queue. Skipping...' in line:
        # [WARN ][org.logstash.common.io.DeadLetterQueueWriter] Event previously submitted to dead letter queue. Skipping...
        return 185

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

r_warn("this can take many many minutes to run, depending on how many and the size of the log files downloaded ...")


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
                        else:
                            res2, date_within_file2 = add_date(line)
                            if res2 == True:
                                if date_within_file2 != date_within_file:
                                    print("Found different dates in same log file !  first=", date_within_file, "second=", date_within_file2, "file:", file)
                                    exit(1)
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
            can_open_gz_file = False
            with gzip.open(file, 'rt') as f:
                can_open_gz_file = True
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
                        else:
                            res2, date_within_file2 = add_date(line)
                            if res2 == True:
                                if date_within_file2 != date_within_file:
                                    print("Found different dates in same log file !  first=", date_within_file, "second=", date_within_file2, "file:", file)
                                    exit(1)
                        if e_num > max_n:
                            max_n = e_num
            
            if can_open_gz_file == False:
                r_die(4, "can't open .gz file:", file)
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

def date_range_list(start_date, end_date):
    # Return list of datetime.date objects (inclusive) between start_date and end_date (inclusive).
    local_date_list = []
    curr_date = start_date
    while curr_date <= end_date:
        local_date_list.append(curr_date)
        curr_date += timedelta(days=1)
    return local_date_list

start_date = date(year=int(date_list[0][0:4]), month=int(date_list[0][5:7]), day=int(date_list[0][8:10]))
stop_date = date(year=int(date_list[-1][0:4]), month=int(date_list[-1][5:7]), day=int(date_list[-1][8:10]))
all_date_list = date_range_list(start_date, stop_date)

all_d = []

# convert to strings to match original format for later checking
for d in all_date_list:
    all_d.append(d.strftime("%Y-%m-%d"))

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
    if i < 100:
        if max_total_counts_widths[i] < 2:
            max_total_counts_widths[i] = 2
    else:
        if max_total_counts_widths[i] < 3:
            max_total_counts_widths[i] = 3

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

missing_days_total = 0

# now print counts by column width in 'date' order
for date in all_d:
    d = parse(date).strftime("%a")
    if d == "Mon":
        r_trace(err_nums)   # show the row column header of the error numbers before every Monday

    if date in date_list:
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
    else:
        missing_day = "Date: " + str(date) + f"{bcolors.WARNING}" + " Missing"+f"{bcolors.ENDC}"
        print(missing_day)
        missing_days_total += 1

print("\nOut of a total span of:", len(all_d), "days there are:")

if missing_days_total == 0:
    print("\n    No missing days.\n")
else:
    print("\n    ", missing_days_total, "days missing.\n")
