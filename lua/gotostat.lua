print("start")

goto test1
::test2::

print("middle")
goto test3

::test1::
goto test2

print("end")
::test3::
print("double end")
