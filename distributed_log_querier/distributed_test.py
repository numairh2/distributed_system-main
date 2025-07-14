import subprocess
import time
import os
envpass = os.environ['ADISHPASS']

def TestPhrase(wordGen, pattern, should_pass):
        # login to each VM and change the log file to read from
        for i in range(1, 11):
                vmNum = str(8500 + i)

                old_filename = "machine."+str(i)+".log" # change with oldfile
                new_filename = "originalText."+str(i)+".log" # new name

                # run a command that renames machine.i.log to originalText.i.log, then generate a randomized logfile
                #   using gen_ran_file.py that saves its result into machine.i.log
                remoteCmds = ["cd ..",
                                "cd ppillai3",
                                f"mv {old_filename} {new_filename}",
                                "cd g85",
                                f"python gen_ran_file.py {i} '{wordGen}'",
                                "exit"]
                remoteCmdsStr = "; ".join(remoteCmds)
                args = ["bash",
                        "-c",
                        f"sshpass -p {envpass} ssh -o StrictHostKeyChecking=no adishj2@fa24-cs425-{vmNum}.cs.illinois.edu '{remoteCmdsStr}'"]
                subprocess.run(args)

        # run caller thru a command that simply runs caller with the given pattern, and saves output into a result variable
        caller_args_string = f"go run main.go caller grep '{pattern}'"
        args = ["bash",
                "-c",
                caller_args_string]
        result = subprocess.run(args,
                        capture_output = True,
                        text=True)
        print("result:")
        print(result.stdout)

        # check result by iterating through every VM and making sure whether or not it should be succeeding, and print accordingly
        valid = True
        for i in range(1,11):
                if f"Line Count machine.{i}.log: 50" not in result.stdout:
                        valid = False
        if (valid and should_pass) or (not valid and not should_pass):
                print("TEST SUCCESS")
        else:
                print("TEST FAILED")

        # login to each VM and switch the original logfile back
        for i in range(1,11):
                vmNum2 = str(8500 + i)

                # run a command that renames machine.i.log to randGen.i.log (since it was created by gen_ran_file.py)
                # Then recover the original data by renaming originalText.i.log into machine.i.log
                old_filename2 = "machine."+str(i)+".log" # change with oldfile
                new_filename2 = "originalText."+str(i)+".log" # new name
                rand_gen_file_name2 = "randgen."+str(i)+".log"
                remoteCmds2 = ["cd ..",
                                "cd ppillai3",
                                f"mv {old_filename2} {rand_gen_file_name2}",
                                f"mv {new_filename2} {old_filename2}",
                                "exit"]
                remoteCmdsStr2 = "; ".join(remoteCmds2)
                
                args2 = ["bash",
                        "-c",
                        f"sshpass -p {envpass} ssh -o StrictHostKeyChecking=no adishj2@fa24-cs425-{vmNum2}.cs.illinois.edu '{remoteCmdsStr2}'"]
                subprocess.run(args2)

# Test Cases
TestPhrase("somethingsomething", "somethingsomething", True) # long string
TestPhrase("helloworld!", "he..oworld!", True) # string with regex
TestPhrase("abcdef", "a.*yyyttyg", False) # string with more complicated regex 1
TestPhrase("abcdef", "a.d.fyyyyy", False) # string with more complicated regex 2
TestPhrase("hello world!", "hello world!", True) # string with space
