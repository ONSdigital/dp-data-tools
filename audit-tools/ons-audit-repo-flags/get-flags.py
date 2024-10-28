# %%
# required python libs:
''' install:
pip3 install beautifulsoup4
pip3 install lxml
pip3 install html5lib
pip3 install requests
pip3 install selenium
pip3 install webdriver-manager
'''
# for this to work, install chromedriver that matches your version of chrome from:
#
# https://chromedriver.chromium.org/downloads
#
# unpack it and:
# mv ~/Downloads/chromedriver /usr/local/bin
# 
# then:
#
# CLOSE ALL google chrome browsers !
#
# In home directory, at command line, do:
#
# mkdir ChromeProfile
#
# then to run chrome in debug port mode, do:
#
# /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222 --user-data-dir="~/ChromeProfile"
#
# In the Google Chrome window that opens up from the above command ...
#  go to "https://github.com/ONSdigital" and SIGN IN
#  (you only need to sign in the first time, including two factor authentication)
# 
# DO NOT CLOSE this window in the python code ... close it manually and kill the command line that launched it with CTRL-C
#
# The window needs to be kept open as the python code is developed / run ...
#
# now develop / run the python code.
#
# When development done, to get final results, at command line do (it takes a few minutes to run):
#
# python3 get-flags.py >repo-settings.csv
#
# Once the script is complete you will need to Quit the Chrome browser that is shown running in the icon bar at bottom of screen.

import os
import selenium
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
import time
import io
from bs4 import BeautifulSoup
import requests
from webdriver_manager.chrome import ChromeDriverManager
import sys

#Install driver
opts=webdriver.ChromeOptions()
opts.headless=False
opts.add_experimental_option("debuggerAddress", "127.0.0.1:9222")

print("Attempting to setup chrome debug port connection ...")

driver = webdriver.Chrome(ChromeDriverManager().install() ,options=opts)
print()

# show title
print("Repo & branch,          require,  required,   dismiss stale,  require,  restrict,  require,  require,   concourse,  concourse,  concourse,   concourse,  require,  require,  include,         restrict,  people,                       allow,   allow")
print(" ,                      pull,     approving,  pull request,   review,   who can,   status,   branches,  -ci/,       -ci/,       -ci/,        -ci/,       signed,   linear,   administrators,  who can,   teams,                        force,   deletions")
print(" ,                      request,  reviews,    approvals,      from,     dismiss,   checks,   to be up,  {repo,      {repo,      {repo,       {repo,      commits,  history,  ,                push to,   or apps,                      pushes,  ")
print(" ,                      reviews,  ,           when new,       code,     pull,      to pass,  to date,   -name},     -name},     -name},      -name},     ,         ,         ,                matching,  with,                         ,        ")
print(" ,                      before,   ,           commits are,    owners,   request,   before,   before,    -audit,     -build,     -component,  -unit,      ,         ,         ,                branches,  push,                         ,        ")
print(" ,                      merging,  ,           pushed,         ,         reviews,   merging,  merging,   ,           ,           ,            ,           ,         ,         ,                ,          access,                       ,        ")

with open("app-list.txt") as f:
    repo_names = f.readlines()
# remove whitespace characters like `\n` at the end of each line
repo_names = [x.strip() for x in repo_names]
# remove empty lines
repo_names = list(filter(None, repo_names))
# remove comment lines
repo_names_clean=[]
for line in repo_names:
    if line[0] != '#':
        repo_names_clean.append(line)

for repo_name in repo_names_clean:

    branches_url = "https://github.com/ONSdigital/" + repo_name + "/settings/branches"
    time.sleep(1) # without delay Github sometimes crashes this app with a timeout
    driver.get(branches_url)

    html = driver.page_source
    soup = BeautifulSoup(html, 'lxml')

    match = soup.find('div', class_='listgroup protected-branches')
    # sanity check that page shows branches
    if match == None:
        print(repo_name)
        print("  ERROR: No 'Branches' created")
        print()
    else:
        branch_names = [i.text for i in soup.findAll('span', {'class':'branch-name flex-self-start css-truncate css-truncate-target'})]
        branch_hrefs = [i.a['href'] for i in soup.findAll('div', {'class':'BtnGroup mt-n5 mt-md-0'})]

        if len(branch_names) != len(branch_hrefs):
            print("Number of Branch Names is not equal to number of Branch hrefs")
            sys.exit()

        for branch_name, branch_href in zip(branch_names, branch_hrefs):
            print(repo_name)
            print("  : ", branch_name)

            search_url = "https://github.com"+branch_href
            time.sleep(1) # without delay Github sometimes crashes this app with a timeout
            driver.get(search_url)
            #print("Waiting for ID to be present ...\n")
            element = WebDriverWait(driver, 3).until(
                EC.presence_of_element_located((By.ID, "has_required_reviews"))  # something that indicates we are on the page we want
            )

            html = driver.page_source
            soup = BeautifulSoup(html, 'lxml')
            match = soup.find('div', class_='js-protected-branch-options')

            line = " ,                      "
            # Require pull request reviews before merging
            state_1 = match.find('div', class_='form-checkbox')
            state_2 = state_1.find('label')
            state_3 = state_2.find('input')
            require_pull_request_reviews_before_merging = "- "
            if 'id' in state_3.attrs:
                if state_3.attrs['id'] == "has_required_reviews":
                    if 'checked' in state_3.attrs:
                        if state_3.attrs['checked'] == "checked":
                            require_pull_request_reviews_before_merging = "on"
            line += require_pull_request_reviews_before_merging

            if require_pull_request_reviews_before_merging == "on":
                # Required approving reviews
                state_1 = match.find('div', class_='require-approving-reviews')
                state_2 = state_1.find('summary', class_='btn')
                required_approving_reviews_count = state_2.span.text
                line += ",       "
                line += required_approving_reviews_count

                # Dismiss stale pull request approvals when new commits are pushed
                state_1 = match.find('div', class_='reviews-dismiss-on-push form-checkbox')
                state_2 = state_1.find('label')
                state_3 = state_2.find('input')
                checked = "- "
                if 'checked' in state_3.attrs:
                    if state_3.attrs['checked'] == "checked":
                        checked = "on"
                line += ",          "
                line += checked

                # Require review from Code Owners
                state_1 = match.find('div', class_='require-code-owner-review form-checkbox')
                state_2 = state_1.find('label')
                state_3 = state_2.find('input')
                checked = "- "
                if 'checked' in state_3.attrs:
                    if state_3.attrs['checked'] == "checked":
                        checked = "on"
                line += ",             "
                line += checked

                # Restrict who can dismiss pull request reviews
                state_1 = match.find('dl', class_='reviews-include-dismiss form-checkbox form-group')
                state_2 = state_1.find('label')
                state_3 = state_2.find('input')
                checked = "- "
                if 'checked' in state_3.attrs:
                    if state_3.attrs['checked'] == "checked":
                        checked = "on"
                line += ",       "
                line += checked

            else:
                line += ",       ,           ,               ,         "

            # unfortunately 3 flags have the same surrounding class name ...
            flags = [i for i in soup.findAll('div', {'class':'js-protected-branch-options protected-branch-options active'})]
            require_status_flag = "- "
            require_branches_up_to_date_flag = "- " # a sub flag
            require_audit_flag = "- " # option of above sub flag
            require_build_flag = "- " # option of above sub flag
            require_component_flag = "- " # option of above sub flag
            require_unit_flag = "- " # option of above sub flag

            allow_force_flag = "- "
            allow_deletions_flag = "- "
            for flag in flags:
                state_1 = flag.find('div', class_='form-checkbox')
                state_2 = state_1.find('label')
                state_3 = state_2.find('input')
                if 'id' in state_3.attrs:
                    if state_3.attrs['id'] == "has_required_statuses":
                        if 'checked' in state_3.attrs:
                            if state_3.attrs['checked'] == "checked":
                                require_status_flag = "on"
                            # Require branches to be up to date before merging
                            state_1 = flag.find('div', class_='js-required-statuses')
                            state_2 = state_1.find('label')
                            state_3a = state_2.find('input')
                            if 'checked' in state_3a.attrs:
                                if state_3a.attrs['checked'] == "checked":
                                    require_branches_up_to_date_flag = "on"

                            checks = [i for i in state_1.findAll('input', {'type':'checkbox'})]
                            for check in checks:
                                # concourse-ci/dp-dataset-api-audit
                                if check.attrs['value'] == "concourse-ci/dp-dataset-api-audit":
                                    if 'checked' in check.attrs:
                                        if check.attrs['checked'] == "checked":
                                            require_audit_flag = "on"
                                # concourse-ci/dp-dataset-api-build
                                if check.attrs['value'] == "concourse-ci/dp-dataset-api-build":
                                    if 'checked' in check.attrs:
                                        if check.attrs['checked'] == "checked":
                                            require_build_flag = "on"
                                # concourse-ci/dp-dataset-api-component
                                if check.attrs['value'] == "concourse-ci/dp-dataset-api-component":
                                    if 'checked' in check.attrs:
                                        if check.attrs['checked'] == "checked":
                                            require_component_flag = "on"
                                # concourse-ci/dp-dataset-api-unit
                                if check.attrs['value'] == "concourse-ci/dp-dataset-api-unit":
                                    if 'checked' in check.attrs:
                                        if check.attrs['checked'] == "checked":
                                            require_unit_flag = "on"

                    # Allow force pushes
                    if state_3.attrs['id'] == "allows_force_pushes":
                        if 'checked' in state_3.attrs:
                            if state_3.attrs['checked'] == "checked":
                                allow_force_flag = "on"
                    # Allow deletions
                    if state_3.attrs['id'] == "allows_deletions":
                        if 'checked' in state_3.attrs:
                            if state_3.attrs['checked'] == "checked":
                                allow_deletions_flag = "on"

            # Require status checks to pass before merging
            line += ",        "
            if require_pull_request_reviews_before_merging != "on":
                line += "  "
            line += require_status_flag

            # Require branches to be up to date before merging
            line += ",       "
            line += require_branches_up_to_date_flag

            # unfortunately 4 flags have the same surrounding class name ...
            flags = [i for i in soup.findAll('div', {'class':'js-protected-branch-options js-toggler-container protected-branch-options active'})]
            require_signed_commits_flag = "- "
            require_linear_history_flag = "- "
            include_administrators_flag = "- "
            restrict_who_can_push_flag = "- "
            restrict_name = "                            " # 28 spaces long
            for flag in flags:
                #state_1 = flag.find('div', class_='form-checkbox')
                state_2 = flag.find('label')
                state_3 = state_2.find('input')
                if 'id' in state_3.attrs:
                    if state_3.attrs['id'] == "has_signature_requirement":
                        if 'checked' in state_3.attrs:
                            if state_3.attrs['checked'] == "checked":
                                require_signed_commits_flag = "on"
                    if state_3.attrs['id'] == "block_merge_commits":
                        if 'checked' in state_3.attrs:
                            if state_3.attrs['checked'] == "checked":
                                require_linear_history_flag = "on"
                    if state_3.attrs['id'] == "enforce_all_for_admins":
                        if 'checked' in state_3.attrs:
                            if state_3.attrs['checked'] == "checked":
                                include_administrators_flag = "on"
                    if state_3.attrs['id'] == "authorized_actors":
                        if 'checked' in state_3.attrs:
                            if state_3.attrs['checked'] == "checked":
                                restrict_who_can_push_flag = "on"
                                na = soup.find('a', {'class':'Link--primary color-text-primary js-protected-branch-pusher'})
                                if na != None:
                                    st = na.find("strong")
                                    restrict_name = st.get_text()

            # concourse-ci/dp-dataset-api-audit
            line += ",        "
            line += require_audit_flag

            # concourse-ci/dp-dataset-api-build
            line += ",         "
            line += require_build_flag

            # concourse-ci/dp-dataset-api-component
            line += ",         "
            line += require_component_flag

            # concourse-ci/dp-dataset-api-unit
            line += ",          "
            line += require_unit_flag

            # Require signed commits
            line += ",         "
            line += require_signed_commits_flag

            # Require linear history
            line += ",       "
            line += require_linear_history_flag

            # Include administrators
            line += ",       "
            line += include_administrators_flag

            # Restrict who can push to matching branches
            line += ",              "
            line += restrict_who_can_push_flag

            # Restrict name
            line += ",        "
            line += restrict_name

            # Allow force pushes
            line += ", "
            line += allow_force_flag

            # Allow deletions
            line += ",      "
            line += allow_deletions_flag

            print(line)

# The following is commented out so as to NOT close the chromedriver that this app is talking to ...
# just incase this app needs to be run again (in an iterative manner for any development changes)
#driver.close()

# %%
